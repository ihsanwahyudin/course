package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fluentpath/api/internal/curriculum"
)

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	cat, err := curriculum.LoadCatalog("../../../../packages/courses")
	if err != nil {
		t.Fatal(err)
	}
	// Svc and Auth are nil: these requests must be rejected before reaching them.
	s := &Server{Catalog: cat, CORSOrigins: []string{"http://localhost:8081"}, Ping: func(context.Context) error { return nil }}
	return s.Router()
}

func TestInputValidation(t *testing.T) {
	h := testRouter(t)
	cases := []struct {
		method, path, body string
		status             int
		code               string
	}{
		// Protected endpoints reject anonymous requests before anything else.
		{"POST", "/api/v1/sessions", `{"course":"english","nodeId":"a0-alphabet-sounds"}`, 401, "unauthorized"},
		{"POST", "/api/v1/sessions/not-a-uuid/steps/0/ask", `{"content":"hi"}`, 401, "unauthorized"},
		{"GET", "/api/v1/me/memory", ``, 401, "unauthorized"},
		{"GET", "/api/v1/notes", ``, 401, "unauthorized"},
		{"GET", "/api/v1/tts?text=hi", ``, 401, "unauthorized"},
		{"GET", "/api/v1/dashboard", ``, 401, "unauthorized"},
		{"POST", "/api/v1/courses/english/enroll", ``, 401, "unauthorized"},
		{"GET", "/api/v1/auth/me", ``, 401, "unauthorized"},
		{"POST", "/api/v1/auth/login", `not json`, 400, "invalid_json"},
		{"POST", "/api/v1/auth/register", `{"email":"a@b.co","extra":1}`, 400, "invalid_json"},
		{"POST", "/api/v1/auth/login", `{"email":"` + strings.Repeat("a", 40<<10) + `"}`, 413, "body_too_large"},
		{"GET", "/api/v1/courses/BAD_ID/roadmap", ``, 404, "course_not_found"},
		{"GET", "/api/v1/courses/english/nodes/BAD_ID", ``, 404, "node_not_found"},
		{"GET", "/api/v1/nope", ``, 404, "not_found"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), `"code":"`+c.code+`"`) {
			t.Errorf("%s %s: got %d %s, want %d %s", c.method, c.path, rec.Code, rec.Body.String(), c.status, c.code)
		}
	}
}

func TestHealthAndCurriculum(t *testing.T) {
	h := testRouter(t)
	for _, path := range []string{"/healthz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/v1/roadmap", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	req.Header.Set("Access-Control-Request-Method", "GET")
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8081" {
		t.Errorf("CORS preflight missing allow-origin: %v", rec.Header())
	}
}

func TestServesWebAppWithSPAFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "_expo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_expo", "a.js"), []byte("js"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, _ := curriculum.LoadCatalog("../../../../packages/courses")
	h := (&Server{Catalog: cat, WebDir: dir, Ping: func(context.Context) error { return nil }}).Router()
	for path, want := range map[string]string{"/": "app", "/roadmap": "app", "/nodes/a0-pronouns": "app", "/_expo/a.js": "js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s: %d %q", path, rec.Code, rec.Body.String())
		}
	}
	// Path traversal never reads outside WebDir.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/../../etc/passwd", nil))
	if strings.Contains(rec.Body.String(), "root:") {
		t.Errorf("path traversal leaked a file outside WebDir")
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/nope", nil))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), "not_found") {
		t.Errorf("unknown API path should stay a JSON 404: %d %s", rec.Code, rec.Body.String())
	}
}

func TestFormPostsAreRejected(t *testing.T) {
	h := testRouter(t)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("email=a&password=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form POST should be refused (CSRF guard), got %d", rec.Code)
	}
}
