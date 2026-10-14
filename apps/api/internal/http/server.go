// Package http exposes the learning service as a JSON API.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/learning"
)

const maxBodyBytes = 32 << 10

type Server struct {
	Svc         *learning.Service
	Cur         *curriculum.Curriculum
	CORSOrigins []string
	Ping        func(context.Context) error
	// Requests per minute per IP; zero means the defaults below.
	RateLimit   int
	AIRateLimit int
	// WebDir, when set, is an `expo export -p web` build served at / so the
	// app and API share one origin (no CORS or local-network prompts).
	WebDir string
}

func (s *Server) limits() (all, ai int) {
	all, ai = 240, 30
	if s.RateLimit > 0 {
		all = s.RateLimit
	}
	if s.AIRateLimit > 0 {
		ai = s.AIRateLimit
	}
	return all, ai
}

func limitExceeded(w http.ResponseWriter, r *http.Request) {
	writeErr(w, &learning.Error{Status: 429, Code: "rate_limited", Message: "Too many requests. Please wait a moment and try again.", Retryable: true})
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: s.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         600,
	}))

	r.Get("/healthz", s.health)
	allLimit, aiLimit := s.limits()

	r.Route("/api/v1", func(r chi.Router) {
		// AI-backed endpoints can take a while; everything else is fast.
		r.Use(middleware.Timeout(90 * time.Second))
		r.Use(httprate.Limit(allLimit, time.Minute, httprate.WithKeyFuncs(httprate.KeyByIP), httprate.WithLimitHandler(limitExceeded)))
		r.Get("/curriculum", s.curriculum)
		r.Get("/roadmap", s.roadmap)
		r.Get("/progress", s.progress)
		r.Get("/nodes/{nodeId}", s.node)
		r.Post("/sessions", s.startSession)
		r.Get("/sessions/{sessionId}", s.getSession)
		r.Group(func(r chi.Router) {
			// Tighter limit on calls that spend AI budget.
			r.Use(httprate.Limit(aiLimit, time.Minute, httprate.WithKeyFuncs(httprate.KeyByIP), httprate.WithLimitHandler(limitExceeded)))
			r.Post("/sessions/{sessionId}/messages", s.sendMessage)
			r.Post("/sessions/{sessionId}/assessment", s.submitAssessment)
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if s.WebDir != "" && r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") {
			s.serveWeb(w, r)
			return
		}
		writeErr(w, &learning.Error{Status: 404, Code: "not_found", Message: "No such endpoint."})
	})
	return r
}

// serveWeb serves static files from WebDir, falling back to index.html so
// client-side routes like /roadmap or /nodes/x load the single-page app.
func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	rel := path.Clean("/" + r.URL.Path)
	full := filepath.Join(s.WebDir, filepath.FromSlash(rel))
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		if strings.HasPrefix(rel, "/_expo/") {
			// Hashed bundle names; safe to cache.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeFile(w, r, full)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filepath.Join(s.WebDir, "index.html"))
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "dur_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	status, db := http.StatusOK, "ok"
	if err := s.Ping(ctx); err != nil {
		status, db = http.StatusServiceUnavailable, "unavailable"
	}
	writeJSON(w, status, map[string]any{"status": http.StatusText(status), "database": db, "curriculumVersion": s.Cur.Version})
}

func (s *Server) curriculum(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"version": s.Cur.Version, "levels": s.Cur.Levels, "nodes": s.Cur.Nodes})
}

func (s *Server) roadmap(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.Roadmap(r.Context())
	respond(w, v, err)
}

func (s *Server) progress(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.Progress(r.Context())
	respond(w, v, err)
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *Server) node(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeId")
	if !idRe.MatchString(id) {
		writeErr(w, &learning.Error{Status: 404, Code: "node_not_found", Message: "No such roadmap node."})
		return
	}
	v, err := s.Svc.Node(r.Context(), id)
	respond(w, v, err)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NodeID string `json:"nodeId"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !idRe.MatchString(body.NodeID) {
		writeErr(w, &learning.Error{Status: 400, Code: "invalid_node_id", Message: "nodeId is required."})
		return
	}
	v, err := s.Svc.StartSession(r.Context(), body.NodeID)
	respond(w, v, err)
}

func sessionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "sessionId")
	if !uuidRe.MatchString(id) {
		writeErr(w, &learning.Error{Status: 404, Code: "session_not_found", Message: "Session not found."})
		return "", false
	}
	return id, true
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.GetSession(r.Context(), id)
	respond(w, v, err)
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.SendMessage(r.Context(), id, body.Content)
	respond(w, v, err)
}

func (s *Server) submitAssessment(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	var body struct {
		Answers []struct {
			ItemID string `json:"itemId"`
			Answer string `json:"answer"`
		} `json:"answers"`
	}
	if !decode(w, r, &body) {
		return
	}
	answers := map[string]string{}
	for _, a := range body.Answers {
		if _, dup := answers[a.ItemID]; dup {
			writeErr(w, &learning.Error{Status: 400, Code: "invalid_answers", Message: "Duplicate answer for " + a.ItemID + "."})
			return
		}
		answers[a.ItemID] = a.Answer
	}
	v, err := s.Svc.SubmitAssessment(r.Context(), id, answers)
	respond(w, v, err)
}

// ---- helpers ----

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeErr(w, &learning.Error{Status: 413, Code: "body_too_large", Message: "Request body is too large."})
		} else {
			writeErr(w, &learning.Error{Status: 400, Code: "invalid_json", Message: "Request body must be valid JSON."})
		}
		return false
	}
	return true
}

func respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func writeErr(w http.ResponseWriter, err error) {
	var le *learning.Error
	if !errors.As(err, &le) {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			le = &learning.Error{Status: 503, Code: "timeout", Message: "The request took too long. Please retry.", Retryable: true}
		} else {
			// Internal details stay in the log, never in the response.
			slog.Error("internal error", "err", err)
			le = &learning.Error{Status: 500, Code: "internal", Message: "Something went wrong on our side.", Retryable: true}
		}
	}
	writeJSON(w, le.Status, map[string]any{"error": le})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
