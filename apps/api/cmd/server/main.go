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
	"strings"
	"syscall"
	"time"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/curriculum"
	apihttp "fluentpath/api/internal/http"
	"fluentpath/api/internal/learning"
	"fluentpath/api/internal/storage"
)

// DemoLearnerID is the single seeded learner for the personal MVP. Keep the
// API private until real authentication exists.
const DemoLearnerID = "00000000-0000-4000-8000-000000000001"

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

	cur, err := curriculum.Load(env("CURRICULUM_DIR", "../../packages/curriculum"))
	if err != nil {
		return err
	}
	slog.Info("curriculum loaded", "version", cur.Version, "nodes", len(cur.Nodes))

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
	if err := store.EnsureLearner(ctx, DemoLearnerID, env("LEARNER_NAME", "Learner")); err != nil {
		return err
	}

	svc := &learning.Service{Store: store, Cur: cur, AI: provider, LearnerID: DemoLearnerID}
	srv := &apihttp.Server{
		Svc: svc, Cur: cur,
		CORSOrigins: strings.Split(env("CORS_ORIGINS", "http://localhost:8081,http://localhost:19006"), ","),
		Ping:        store.Pool.Ping,
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
		return ai.NewOpenAI(key, model, os.Getenv("OPENAI_EVAL_MODEL"), os.Getenv("OPENAI_BASE_URL")), nil
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
