// Command server runs the FluentPath API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/auth"
	"fluentpath/api/internal/curriculum"
	apihttp "fluentpath/api/internal/http"
	"fluentpath/api/internal/learning"
	"fluentpath/api/internal/storage"
	"fluentpath/api/internal/tutor"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	loadDotEnv(".env")
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	catalog, err := curriculum.LoadCatalog(env("COURSES_DIR", "../../packages/courses"))
	if err != nil {
		return err
	}
	for _, c := range catalog.Courses {
		// Every ready lesson ships with reviewed, hand-written steps.
		if err := tutor.ValidateAuthored(c); err != nil {
			return fmt.Errorf("course %q lesson steps: %w", c.Slug, err)
		}
		slog.Info("course loaded", "slug", c.Slug, "version", c.Version, "nodes", len(c.Nodes), "status", c.Status)
	}
	// Personalisation (onboarding questions, goal badges and targets, tutor
	// hints) is switched off for now; PERSONALIZATION=on brings it back.
	if env("PERSONALIZATION", "off") != "on" {
		catalog.DisableOnboarding()
	}

	provider, err := newProvider()
	if err != nil {
		return err
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	store, err := storage.Open(connectCtx, dbURL, "")
	cancel()
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	go cleanExpiredSessions(ctx, store)

	svc := &learning.Service{Store: store, Catalog: catalog, AI: provider, TimeZone: env("APP_TIMEZONE", "Asia/Jakarta"),
		DailyTokens: envInt("AI_DAILY_TOKENS_PER_USER", 60000),
		// Pre-make lesson audio when the provider has a voice (WARM_SPEECH=off to disable).
		WarmSpeech: provider.Name() != "mock" && env("WARM_SPEECH", "on") != "off",
		// Kept across restarts so lesson audio is only ever made once.
		SpeechCacheDir: env("SPEECH_CACHE_DIR", "data/tts")}
	svc.WarmAllLessons()
	srv := &apihttp.Server{
		Svc: svc, Auth: &auth.Service{Store: store}, Catalog: catalog,
		CORSOrigins:   strings.Split(env("CORS_ORIGINS", "http://localhost:8081,http://localhost:19006"), ","),
		Ping:          store.Pool.Ping,
		RateLimit:     envInt("RATE_LIMIT_PER_MIN", 0),
		AIRateLimit:   envInt("AI_RATE_LIMIT_PER_MIN", 0),
		AuthRateLimit: envInt("AUTH_RATE_LIMIT_PER_MIN", 0),
		WebDir:        os.Getenv("WEB_DIR"),
	}
	if srv.WebDir != "" {
		if _, err := os.Stat(srv.WebDir + "/index.html"); err != nil {
			return fmt.Errorf("WEB_DIR %q has no index.html (run `npx expo export -p web` in apps/mobile): %w", srv.WebDir, err)
		}
		slog.Info("serving web app", "dir", srv.WebDir)
	}
	httpSrv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", httpSrv.Addr, "ai", provider.Name())
		errCh <- httpSrv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}

// cleanExpiredSessions removes expired login sessions and AI cache entries
// once an hour.
func cleanExpiredSessions(ctx context.Context, store *storage.Store) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if n, err := storage.DeleteExpiredAuthSessions(ctx, store.Pool); err != nil && ctx.Err() == nil {
			slog.Warn("cleaning expired sessions failed", "err", err)
		} else if n > 0 {
			slog.Info("removed expired sessions", "count", n)
		}
		if n, err := storage.DeleteExpiredCache(ctx, store.Pool); err != nil && ctx.Err() == nil {
			slog.Warn("cleaning the AI cache failed", "err", err)
		} else if n > 0 {
			slog.Info("removed expired AI cache entries", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func newProvider() (ai.Provider, error) {
	switch p := env("AI_PROVIDER", "mock"); p {
	case "mock":
		slog.Warn("using the deterministic mock AI provider")
		return &ai.Mock{}, nil
	case "openai":
		key, model := os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL")
		if key == "" || model == "" {
			return nil, errors.New("AI_PROVIDER=openai needs OPENAI_API_KEY and OPENAI_MODEL")
		}
		o := ai.NewOpenAI(key, model, os.Getenv("OPENAI_EVAL_MODEL"), os.Getenv("OPENAI_BASE_URL"))
		if v := os.Getenv("OPENAI_TTS_MODEL"); v != "" {
			o.TTSModel = v
		}
		if v := os.Getenv("OPENAI_TTS_VOICE"); v != "" {
			o.TTSVoice = v
		}
		if v := os.Getenv("OPENAI_TRANSCRIBE_MODEL"); v != "" {
			o.TranscribeModel = v
		}
		return o, nil
	default:
		return nil, fmt.Errorf("unknown AI_PROVIDER %q (want openai or mock)", p)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

// loadDotEnv sets variables from a simple KEY=VALUE file without overriding
// the real environment. Missing file is fine.
func loadDotEnv(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
}
