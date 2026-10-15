// Package http exposes accounts and the learning service as a JSON API, and
// can also serve the exported web app from the same origin.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
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
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept-Language", nativeClientHeader},
		AllowCredentials: true,
		MaxAge:           600,
	}))

	r.Get("/healthz", s.health)
	allLimit, aiLimit, authLimit := s.limits()

	r.Route("/api/v1", func(r chi.Router) {
		// Writing a lesson plan can take a while on the first visit.
		r.Use(middleware.Timeout(170 * time.Second))
		r.Use(perIP(allLimit))
		// JSON-only bodies (audio for /transcribe): with SameSite=Lax cookies
		// this blocks cross-site form CSRF, since forms can't send these types.
		r.Use(allowBodies)
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
			r.Get("/sessions/{sessionId}/steps/{step}/listen", s.stepListen)
			r.Get("/sessions/{sessionId}/assessment/{itemId}/listen", s.itemListen)
			r.Get("/me/memory", s.memory)
			r.Patch("/me/memory", s.setMemory)
			r.Delete("/me/memory", s.deleteAllMemory)
			r.Patch("/me/memory/{memoryId}", s.updateMemory)
			r.Delete("/me/memory/{memoryId}", s.deleteMemory)
			r.Get("/notes", s.notes)
			r.Post("/notes", s.saveNote)
			r.Delete("/notes/{noteId}", s.deleteNote)
			r.Post("/flashcards", s.saveFlashcards)
			r.Get("/flashcards/due", s.dueFlashcards)
			r.Post("/flashcards/{cardId}/review", s.reviewFlashcard)
			r.Delete("/flashcards/{cardId}", s.deleteFlashcard)
			r.Group(func(r chi.Router) {
				// Tighter limit on calls that spend AI budget.
				r.Use(perIP(aiLimit))
				r.Post("/sessions/{sessionId}/steps/{step}/next", s.nextStep)
				r.Post("/sessions/{sessionId}/steps/{step}/answer", s.answerStep)
				r.Post("/sessions/{sessionId}/steps/{step}/hint", s.hintStep)
				r.Post("/sessions/{sessionId}/steps/{step}/reveal", s.revealStep)
				r.Post("/sessions/{sessionId}/steps/{step}/similar", s.similarStep)
				r.Post("/sessions/{sessionId}/steps/{step}/ask", s.askStep)
				r.Post("/sessions/{sessionId}/assessment", s.submitAssessment)
				r.Get("/tts", s.speech)
				r.Post("/transcribe", s.transcribe)
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

// allowBodies accepts JSON bodies everywhere and audio for /transcribe.
func allowBodies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 || r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ct := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
		if ct == "application/json" || (strings.HasSuffix(r.URL.Path, "/transcribe") && strings.HasPrefix(ct, "audio/")) {
			next.ServeHTTP(w, r)
			return
		}
		writeErr(w, &apperr.Error{Status: 415, Code: "unsupported_media_type", Message: "Send JSON."})
	})
}

// ---- lesson steps ----

func stepParams(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	id, ok := sessionID(w, r)
	if !ok {
		return "", 0, false
	}
	n, err := strconv.Atoi(chi.URLParam(r, "step"))
	if err != nil || n < 0 || n > 100 {
		writeErr(w, &apperr.Error{Status: 404, Code: "step_not_open", Message: "That step isn't open yet."})
		return "", 0, false
	}
	return id, n, true
}

type similarFlag struct {
	Similar bool `json:"similar"`
}

// optionalBody decodes a JSON body if there is one.
func optionalBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.ContentLength == 0 {
		return true
	}
	return decode(w, r, v)
}

func (s *Server) nextStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.NextStep(r.Context(), userID(r), id, n)
	respond(w, v, err)
}

func (s *Server) answerStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	var body learning.AnswerInput
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.AnswerStep(r.Context(), userID(r), id, n, body)
	respond(w, v, err)
}

func (s *Server) hintStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	var body similarFlag
	if !optionalBody(w, r, &body) {
		return
	}
	v, err := s.Svc.HintStep(r.Context(), userID(r), id, n, body.Similar)
	respond(w, v, err)
}

func (s *Server) revealStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	var body similarFlag
	if !optionalBody(w, r, &body) {
		return
	}
	v, err := s.Svc.RevealStep(r.Context(), userID(r), id, n, body.Similar)
	respond(w, v, err)
}

func (s *Server) similarStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	v, err := s.Svc.SimilarStep(r.Context(), userID(r), id, n)
	respond(w, v, err)
}

func (s *Server) askStep(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	var body learning.AskInput
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.AskStep(r.Context(), userID(r), id, n, body)
	respond(w, v, err)
}

// ---- speech ----

// writeListen sends a listening question's audio, or (no recording yet) its
// text as JSON for the device's voice.
func writeListen(w http.ResponseWriter, a learning.ListenAudio, err error) {
	if err != nil {
		writeErr(w, err)
		return
	}
	if a.Audio == nil {
		w.Header().Set("X-Speech-Fallback", "device")
		respond(w, map[string]string{"text": a.Text}, nil)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "private, max-age=604800")
	_, _ = w.Write(a.Audio)
}

func (s *Server) stepListen(w http.ResponseWriter, r *http.Request) {
	id, n, ok := stepParams(w, r)
	if !ok {
		return
	}
	a, err := s.Svc.StepListen(r.Context(), userID(r), id, n)
	writeListen(w, a, err)
}

func (s *Server) itemListen(w http.ResponseWriter, r *http.Request) {
	id, ok := sessionID(w, r)
	if !ok {
		return
	}
	a, err := s.Svc.ItemListen(r.Context(), userID(r), id, chi.URLParam(r, "itemId"))
	writeListen(w, a, err)
}

func (s *Server) speech(w http.ResponseWriter, r *http.Request) {
	audio, err := s.Svc.Speech(r.Context(), userID(r), r.URL.Query().Get("text"))
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "private, max-age=604800")
	_, _ = w.Write(audio)
}

var audioExt = map[string]string{"audio/webm": "webm", "audio/ogg": "ogg", "audio/mp4": "m4a", "audio/m4a": "m4a", "audio/x-m4a": "m4a",
	"audio/mpeg": "mp3", "audio/wav": "wav", "audio/x-wav": "wav", "audio/aac": "aac"}

func (s *Server) transcribe(w http.ResponseWriter, r *http.Request) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	ext, ok := audioExt[ct]
	if !ok {
		writeErr(w, &apperr.Error{Status: 415, Code: "unsupported_audio", Message: "Unsupported audio format."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, learning.MaxAudioBytes)
	audio, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, &apperr.Error{Status: 413, Code: "body_too_large", Message: "Recordings must be under 2 MB (about a minute)."})
		return
	}
	text, err := s.Svc.Transcribe(r.Context(), userID(r), audio, "recording."+ext)
	respond(w, map[string]string{"text": text}, err)
}

// ---- memory & notes ----

func (s *Server) memory(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.Memory(r.Context(), userID(r))
	respond(w, v, err)
}

func (s *Server) setMemory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.SetMemoryEnabled(r.Context(), userID(r), body.Enabled)
	respond(w, v, err)
}

func (s *Server) deleteAllMemory(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.DeleteAllMemory(r.Context(), userID(r))
	respond(w, v, err)
}

func memoryID(w http.ResponseWriter, r *http.Request, param, code string) (string, bool) {
	id := chi.URLParam(r, param)
	if !uuidRe.MatchString(id) {
		writeErr(w, &apperr.Error{Status: 404, Code: code, Message: "Not found."})
		return "", false
	}
	return id, true
}

func (s *Server) updateMemory(w http.ResponseWriter, r *http.Request) {
	id, ok := memoryID(w, r, "memoryId", "memory_not_found")
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.UpdateMemory(r.Context(), userID(r), id, body.Text)
	respond(w, v, err)
}

func (s *Server) deleteMemory(w http.ResponseWriter, r *http.Request) {
	id, ok := memoryID(w, r, "memoryId", "memory_not_found")
	if !ok {
		return
	}
	v, err := s.Svc.DeleteMemory(r.Context(), userID(r), id)
	respond(w, v, err)
}

func (s *Server) notes(w http.ResponseWriter, r *http.Request) {
	v, err := s.Svc.Notes(r.Context(), userID(r))
	respond(w, map[string]any{"notes": v}, err)
}

func (s *Server) saveNote(w http.ResponseWriter, r *http.Request) {
	var body learning.NoteInput
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.SaveNote(r.Context(), userID(r), body)
	respond(w, v, err)
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	id, ok := memoryID(w, r, "noteId", "note_not_found")
	if !ok {
		return
	}
	respond(w, map[string]bool{"ok": true}, s.Svc.DeleteNote(r.Context(), userID(r), id))
}

// ---- flashcards ----

func (s *Server) saveFlashcards(w http.ResponseWriter, r *http.Request) {
	var body learning.FlashcardInput
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.SaveFlashcards(r.Context(), userID(r), body)
	respond(w, v, err)
}

func (s *Server) dueFlashcards(w http.ResponseWriter, r *http.Request) {
	course := r.URL.Query().Get("course")
	if course != "" && !idRe.MatchString(course) {
		writeErr(w, &apperr.Error{Status: 400, Code: "invalid_course", Message: "Invalid course."})
		return
	}
	v, err := s.Svc.DueFlashcards(r.Context(), userID(r), course)
	respond(w, v, err)
}

func flashcardID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "cardId")
	if !uuidRe.MatchString(id) {
		writeErr(w, &apperr.Error{Status: 404, Code: "flashcard_not_found", Message: "Flashcard not found."})
		return "", false
	}
	return id, true
}

func (s *Server) reviewFlashcard(w http.ResponseWriter, r *http.Request) {
	id, ok := flashcardID(w, r)
	if !ok {
		return
	}
	var body struct {
		Grade string `json:"grade"`
	}
	if !decode(w, r, &body) {
		return
	}
	v, err := s.Svc.ReviewFlashcard(r.Context(), userID(r), id, body.Grade)
	respond(w, v, err)
}

func (s *Server) deleteFlashcard(w http.ResponseWriter, r *http.Request) {
	id, ok := flashcardID(w, r)
	if !ok {
		return
	}
	respond(w, map[string]bool{"ok": true}, s.Svc.DeleteFlashcard(r.Context(), userID(r), id))
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
