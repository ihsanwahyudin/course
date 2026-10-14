// Package http exposes accounts and the learning service as a JSON API, and
// can also serve the exported web app from the same origin.
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

	"fluentpath/api/internal/apperr"
	"fluentpath/api/internal/auth"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/learning"
	"fluentpath/api/internal/storage"
)

const (
	maxBodyBytes = 32 << 10
	cookieName   = "fp_session"
	// Native clients send this header to receive the bearer token in the
	// response body; web clients only ever get the HttpOnly cookie.
	nativeClientHeader = "X-FluentPath-Client"
)

type Server struct {
	Svc         *learning.Service
	Auth        *auth.Service
	Catalog     *curriculum.Catalog
	CORSOrigins []string
	Ping        func(context.Context) error
	// Requests per minute per IP; zero means the defaults in limits().
	RateLimit     int
	AIRateLimit   int
	AuthRateLimit int
	// WebDir, when set, is an `expo export -p web` build served at / so the
	// app and API share one origin (no CORS or local-network prompts).
	WebDir string
}

func (s *Server) limits() (all, ai, login int) {
	all, ai, login = 240, 30, 10
	if s.RateLimit > 0 {
		all = s.RateLimit
	}
	if s.AIRateLimit > 0 {
		ai = s.AIRateLimit
	}
	if s.AuthRateLimit > 0 {
		login = s.AuthRateLimit
	}
	return
}

func limitExceeded(w http.ResponseWriter, r *http.Request) {
	writeErr(w, &apperr.Error{Status: 429, Code: "rate_limited", Message: "Too many requests. Please wait a moment and try again.", Retryable: true})
}

func perIP(n int) func(http.Handler) http.Handler {
	return httprate.Limit(n, time.Minute, httprate.WithKeyFuncs(httprate.KeyByIP), httprate.WithLimitHandler(limitExceeded))
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept-Language", nativeClientHeader},
		AllowCredentials: true,
		MaxAge:           600,
	}))

	r.Get("/healthz", s.health)
	allLimit, aiLimit, authLimit := s.limits()

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Timeout(90 * time.Second))
		r.Use(perIP(allLimit))
		// JSON-only POSTs: with SameSite=Lax cookies this blocks cross-site form CSRF.
		r.Use(middleware.AllowContentType("application/json"))
		r.Use(s.identify)
		r.Use(withLocale)

		r.Route("/auth", func(r chi.Router) {
			r.With(perIP(authLimit)).Post("/register", s.register)
			r.With(perIP(authLimit)).Post("/login", s.login)
			r.Post("/logout", s.logout)
			r.With(requireUser).Get("/me", s.me)
			r.With(requireUser).Patch("/me", s.updateMe)
		})

		// Public (optional auth): catalog and course roadmaps.
		r.Get("/courses", s.courses)
		r.Get("/courses/{course}/roadmap", s.roadmap)
		r.Get("/courses/{course}/nodes/{nodeId}", s.node)

		r.Group(func(r chi.Router) {
			r.Use(requireUser)
			r.Post("/courses/{course}/enroll", s.enroll)
			r.Get("/courses/{course}/profile", s.getProfile)
			r.Put("/courses/{course}/profile", s.saveProfile)
			r.Get("/courses/{course}/progress", s.courseProgress)
			r.Get("/dashboard", s.dashboard)
			r.Post("/sessions", s.startSession)
			r.Get("/sessions/{sessionId}", s.getSession)
			r.Group(func(r chi.Router) {
				// Tighter limit on calls that spend AI budget.
				r.Use(perIP(aiLimit))
				r.Post("/sessions/{sessionId}/messages", s.sendMessage)
				r.Post("/sessions/{sessionId}/assessment", s.submitAssessment)
			})
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if s.WebDir != "" && r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") {
			s.serveWeb(w, r)
			return
		}
		writeErr(w, &apperr.Error{Status: 404, Code: "not_found", Message: "No such endpoint."})
	})
	return r
}

// serveWeb serves static files from WebDir, falling back to index.html so
// client-side routes like /dashboard or /courses/x load the single-page app.
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
	writeJSON(w, status, map[string]any{"status": http.StatusText(status), "database": db, "courses": len(s.Catalog.Courses)})
}

// ---- identity ----

type ctxKey struct{}

type identity struct {
	user  storage.User
	token string
}

func tokenFrom(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

// identify attaches the logged-in user, if any. Invalid or expired tokens
// are treated as anonymous; requireUser turns that into a 401.
func (s *Server) identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := tokenFrom(r)
		if token == "" || s.Auth == nil {
			next.ServeHTTP(w, r)
			return
		}
		u, err := s.Auth.Authenticate(r.Context(), token)
		if err != nil {
			var ae *apperr.Error
			if !errors.As(err, &ae) {
				writeErr(w, err)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, identity{u, token})))
	})
}

func currentUser(r *http.Request) (storage.User, bool) {
	id, ok := r.Context().Value(ctxKey{}).(identity)
	return id.user, ok
}

func userID(r *http.Request) string {
	u, _ := currentUser(r)
	return u.ID
}

func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentUser(r); !ok {
			writeErr(w, apperr.Unauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requestLang picks "id" or "en" from Accept-Language (first tag wins);
// "" when the header names neither.
func requestLang(header string) string {
	first := strings.ToLower(strings.TrimSpace(strings.Split(header, ",")[0]))
	first = strings.TrimSpace(strings.Split(first, ";")[0])
	switch {
	case strings.HasPrefix(first, "id"), strings.HasPrefix(first, "in"):
		return "id"
	case strings.HasPrefix(first, "en"):
		return "en"
	}
	return ""
}

// withLocale sets the UI language for the request: Accept-Language first,
// then the signed-in user's saved preference, then English.
func withLocale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := requestLang(r.Header.Get("Accept-Language"))
		if lang == "" {
			if u, ok := currentUser(r); ok {
				lang = u.Locale
			}
		}
		if lang == "" {
			lang = "en"
		}
		w.Header().Set("Content-Language", lang)
		next.ServeHTTP(w, r.WithContext(learning.WithLang(r.Context(), lang)))
	})
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: secureRequest(r), SameSite: http.SameSiteLaxMode,
	})
}

type authResponse struct {
	User  storage.User `json:"user"`
	Token string       `json:"token,omitempty"`
}

func (s *Server) finishLogin(w http.ResponseWriter, r *http.Request, u storage.User, token string, err error) {
	if err != nil {
		writeErr(w, err)
		return
	}
	setSessionCookie(w, r, token, int(auth.SessionTTL.Seconds()))
	resp := authResponse{User: u}
	if r.Header.Get(nativeClientHeader) == "native" {
		resp.Token = token
	}
	writeJSON(w, 200, resp)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if !decode(w, r, &body) {
		return
	}
	reg := auth.Registration{Email: body.Email, Password: body.Password, DisplayName: body.DisplayName, Locale: requestLang(r.Header.Get("Accept-Language"))}
	u, token, err := s.Auth.Register(r.Context(), reg, r.UserAgent())
	s.finishLogin(w, r, u, token, err)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &body) {
		return
	}
	u, token, err := s.Auth.Login(r.Context(), body.Email, body.Password, r.UserAgent())
	s.finishLogin(w, r, u, token, err)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.Auth.Logout(r.Context(), tokenFrom(r)); err != nil {
		writeErr(w, err)
		return
	}
	setSessionCookie(w, r, "", -1)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	writeJSON(w, 200, authResponse{User: u})
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Locale string `json:"locale"`
	}
	if !decode(w, r, &body) {
		return
	}
	u, err := s.Auth.SetLocale(r.Context(), userID(r), body.Locale)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, 200, authResponse{User: u})
}

// ---- courses ----

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func courseParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	c := chi.URLParam(r, "course")
	if !idRe.MatchString(c) {
		writeErr(w, &apperr.Error{Status: 404, Code: "course_not_found", Message: "No such course."})
		return "", false
	}
	return c, true
}

func (s *Server) courses(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.Courses(r.Context(), userID(r))
	respond(w, map[string]any{"courses": v}, err)
}

func (s *Server) roadmap(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.Roadmap(r.Context(), userID(r), course)
	respond(w, v, err)
}

func (s *Server) node(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "nodeId")
	if !idRe.MatchString(id) {
		writeErr(w, &apperr.Error{Status: 404, Code: "node_not_found", Message: "No such roadmap node."})
		return
	}
	v, err := s.Svc.Node(r.Context(), userID(r), course, id)
	respond(w, v, err)
}

func (s *Server) enroll(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.Enroll(r.Context(), userID(r), course)
	respond(w, v, err)
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.GetProfile(r.Context(), userID(r), course)
	respond(w, v, err)
}

func (s *Server) saveProfile(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	var body learning.ProfileInput
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.SaveProfile(r.Context(), userID(r), course, body)
	respond(w, v, err)
}

func (s *Server) courseProgress(w http.ResponseWriter, r *http.Request) {
	course, ok := courseParam(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.CourseProgress(r.Context(), userID(r), course)
	respond(w, v, err)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	v, err := s.Svc.Dashboard(r.Context(), u)
	respond(w, v, err)
}

// ---- lesson sessions ----

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Course string `json:"course"`
		NodeID string `json:"nodeId"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !idRe.MatchString(body.Course) || !idRe.MatchString(body.NodeID) {
		writeErr(w, &apperr.Error{Status: 400, Code: "invalid_node_id", Message: "course and nodeId are required."})
		return
	}
	v, err := s.Svc.StartSession(r.Context(), userID(r), body.Course, body.NodeID)
	respond(w, v, err)
}

func sessionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "sessionId")
	if !uuidRe.MatchString(id) {
		writeErr(w, &apperr.Error{Status: 404, Code: "session_not_found", Message: "Session not found."})
		return "", false
	}
	return id, true
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.GetSession(r.Context(), userID(r), id)
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
	v, err := s.Svc.SendMessage(r.Context(), userID(r), id, body.Content)
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
			writeErr(w, &apperr.Error{Status: 400, Code: "invalid_answers", Message: "Duplicate answer for " + a.ItemID + "."})
			return
		}
		answers[a.ItemID] = a.Answer
	}
	v, err := s.Svc.SubmitAssessment(r.Context(), userID(r), id, answers)
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
			writeErr(w, &apperr.Error{Status: 413, Code: "body_too_large", Message: "Request body is too large."})
		} else {
			writeErr(w, &apperr.Error{Status: 400, Code: "invalid_json", Message: "Request body must be valid JSON."})
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
	var le *apperr.Error
	if !errors.As(err, &le) {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			le = &apperr.Error{Status: 503, Code: "timeout", Message: "The request took too long. Please retry.", Retryable: true}
		} else {
			// Internal details stay in the log, never in the response.
			slog.Error("internal error", "err", err)
			le = &apperr.Error{Status: 500, Code: "internal", Message: "Something went wrong on our side.", Retryable: true}
		}
	}
	writeJSON(w, le.Status, map[string]any{"error": le})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
