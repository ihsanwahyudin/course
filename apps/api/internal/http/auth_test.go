package http

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"fluentpath/api/internal/ai"
	"fluentpath/api/internal/auth"
	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/learning"
	"fluentpath/api/internal/storage"
)

// dbRouter is a full router backed by a fresh schema in TEST_DATABASE_URL.
func dbRouter(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", rand.Uint64())
	admin, err := storage.Open(ctx, url, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(ctx, url, schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store.Close()
		_, _ = admin.Pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cat, err := curriculum.LoadCatalog("../../../../packages/courses")
	if err != nil {
		t.Fatal(err)
	}
	svc := &learning.Service{Store: store, Catalog: cat, AI: &ai.Mock{}, TimeZone: "UTC"}
	srv := &Server{Svc: svc, Auth: &auth.Service{Store: store}, Catalog: cat, Ping: store.Pool.Ping, AuthRateLimit: 1000}
	return srv.Router(), store
}

type call struct {
	method, path, body string
	cookie             *http.Cookie
	bearer             string
	native             bool
}

func do(h http.Handler, c call) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
	req.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.native {
		req.Header.Set(nativeClientHeader, "native")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	return nil
}

func errCode(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	s, _ := e["code"].(string)
	return s
}

func TestAuthFlow(t *testing.T) {
	h, _ := dbRouter(t)
	reg := `{"email":"Rina@Example.com","password":"correct horse","displayName":"Rina"}`

	rec, body := do(h, call{method: "POST", path: "/api/v1/auth/register", body: reg})
	if rec.Code != 200 {
		t.Fatalf("register: %d %v", rec.Code, body)
	}
	if _, has := body["token"]; has {
		t.Fatal("web clients must not receive the token in the body")
	}
	ck := sessionCookie(rec)
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected HttpOnly SameSite=Lax session cookie, got %+v", ck)
	}
	if u := body["user"].(map[string]any); u["email"] != "rina@example.com" {
		t.Fatalf("email should be normalised: %v", u["email"])
	}

	if rec, body = do(h, call{method: "GET", path: "/api/v1/auth/me", cookie: ck}); rec.Code != 200 {
		t.Fatalf("me with cookie: %d %v", rec.Code, body)
	}
	if rec, body = do(h, call{method: "POST", path: "/api/v1/auth/register", body: reg}); errCode(body) != "email_taken" {
		t.Fatalf("duplicate register: %d %v", rec.Code, body)
	}
	if _, body = do(h, call{method: "POST", path: "/api/v1/auth/login", body: `{"email":"rina@example.com","password":"wrong pass"}`}); errCode(body) != "invalid_credentials" {
		t.Fatalf("wrong password: %v", body)
	}
	if _, body = do(h, call{method: "POST", path: "/api/v1/auth/login", body: `{"email":"nobody@example.com","password":"whatever1"}`}); errCode(body) != "invalid_credentials" {
		t.Fatalf("unknown email must look like a wrong password: %v", body)
	}

	// Native login gets a bearer token that works without the cookie.
	rec, body = do(h, call{method: "POST", path: "/api/v1/auth/login", body: `{"email":"rina@example.com","password":"correct horse"}`, native: true})
	token, _ := body["token"].(string)
	if rec.Code != 200 || token == "" {
		t.Fatalf("native login: %d %v", rec.Code, body)
	}
	if rec, _ = do(h, call{method: "GET", path: "/api/v1/dashboard", bearer: token}); rec.Code != 200 {
		t.Fatalf("dashboard with bearer: %d", rec.Code)
	}

	// Logout revokes that session only.
	if rec, _ = do(h, call{method: "POST", path: "/api/v1/auth/logout", bearer: token}); rec.Code != 200 {
		t.Fatalf("logout: %d", rec.Code)
	}
	if _, body = do(h, call{method: "GET", path: "/api/v1/auth/me", bearer: token}); errCode(body) != "unauthorized" {
		t.Fatalf("token should be revoked after logout: %v", body)
	}
	if rec, _ = do(h, call{method: "GET", path: "/api/v1/auth/me", cookie: ck}); rec.Code != 200 {
		t.Fatalf("other session should survive: %d", rec.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	h, _ := dbRouter(t)
	for body, want := range map[string]string{
		`{"email":"not-an-email","password":"longenough","displayName":"A"}`:    "invalid_email",
		`{"email":"a@b","password":"longenough","displayName":"A"}`:             "invalid_email",
		`{"email":"a@b.co","password":"short","displayName":"A"}`:               "invalid_password",
		`{"email":"a@b.co","password":"longenough","displayName":"   "}`:        "invalid_name",
		`{"email":"Name <a@b.co>","password":"longenough","displayName":"Ann"}`: "invalid_email",
	} {
		if _, got := do(h, call{method: "POST", path: "/api/v1/auth/register", body: body}); errCode(got) != want {
			t.Errorf("%s: got %v, want %s", body, got, want)
		}
	}
}

func TestExpiredSessionIsRejected(t *testing.T) {
	h, store := dbRouter(t)
	rec, _ := do(h, call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"old@example.com","password":"longenough","displayName":"Old"}`})
	ck := sessionCookie(rec)
	if _, err := store.Pool.Exec(context.Background(), `UPDATE auth_sessions SET expires_at = NOW() - INTERVAL '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if _, body := do(h, call{method: "GET", path: "/api/v1/dashboard", cookie: ck}); errCode(body) != "unauthorized" {
		t.Fatalf("expired session should be refused: %v", body)
	}
}

func TestPublicCatalogAndPerUserData(t *testing.T) {
	h, _ := dbRouter(t)
	rec, body := do(h, call{method: "GET", path: "/api/v1/courses/english/roadmap"})
	if rec.Code != 200 || body["authenticated"] != false {
		t.Fatalf("anonymous roadmap: %d %v", rec.Code, body)
	}
	a := sessionCookie(func() *httptest.ResponseRecorder {
		r, _ := do(h, call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"a@example.com","password":"longenough","displayName":"A"}`})
		return r
	}())
	b := sessionCookie(func() *httptest.ResponseRecorder {
		r, _ := do(h, call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"b@example.com","password":"longenough","displayName":"B"}`})
		return r
	}())
	rec, body = do(h, call{method: "POST", path: "/api/v1/sessions", body: `{"course":"english","nodeId":"a0-alphabet-sounds"}`, cookie: a})
	if rec.Code != 200 {
		t.Fatalf("start session: %d %v", rec.Code, body)
	}
	id := body["id"].(string)
	if _, body = do(h, call{method: "GET", path: "/api/v1/sessions/" + id, cookie: b}); errCode(body) != "session_not_found" {
		t.Fatalf("user B read user A's session: %v", body)
	}
	_, body = do(h, call{method: "GET", path: "/api/v1/dashboard", cookie: a})
	if courses := body["courses"].([]any); len(courses) != 1 {
		t.Fatalf("starting a lesson should enrol user A: %v", body["courses"])
	}
	_, body = do(h, call{method: "GET", path: "/api/v1/dashboard", cookie: b})
	if courses := body["courses"].([]any); len(courses) != 0 {
		t.Fatalf("user B should have no courses: %v", body["courses"])
	}
}

func doLang(h http.Handler, c call, lang string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
	req.Header.Set("Content-Type", "application/json")
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

func TestLocalizedResponsesAndSavedLocale(t *testing.T) {
	h, _ := dbRouter(t)
	path := "/api/v1/courses/english/nodes/a0-pronouns"
	if _, body := doLang(h, call{method: "GET", path: path}, "id-ID,id;q=0.9"); body["title"] != "Kata Ganti Orang" {
		t.Fatalf("Accept-Language id should give Indonesian, got %v", body["title"])
	}
	if _, body := doLang(h, call{method: "GET", path: path}, "en-US"); body["title"] != "Personal Pronouns" {
		t.Fatalf("en should give English, got %v", body["title"])
	}
	if _, body := doLang(h, call{method: "GET", path: path}, ""); body["title"] != "Personal Pronouns" {
		t.Fatalf("default should be English, got %v", body["title"])
	}

	// Registering from an Indonesian browser saves "id"; requests without a
	// header then fall back to the saved preference.
	rec, body := doLang(h, call{method: "POST", path: "/api/v1/auth/register", body: `{"email":"sari@example.com","password":"longenough","displayName":"Sari"}`}, "id")
	if u := body["user"].(map[string]any); u["locale"] != "id" {
		t.Fatalf("locale at sign-up should be saved: %v", u["locale"])
	}
	ck := sessionCookie(rec)
	if _, body = doLang(h, call{method: "GET", path: "/api/v1/courses", cookie: ck}, ""); body["courses"].([]any)[0].(map[string]any)["title"] != "Bahasa Inggris" {
		t.Fatalf("saved locale should apply without a header: %v", body["courses"])
	}

	if _, body = doLang(h, call{method: "PATCH", path: "/api/v1/auth/me", body: `{"locale":"fr"}`, cookie: ck}, ""); errCode(body) != "invalid_locale" {
		t.Fatalf("unsupported locale must be rejected: %v", body)
	}
	if _, body = doLang(h, call{method: "PATCH", path: "/api/v1/auth/me", body: `{"locale":"en"}`, cookie: ck}, ""); body["user"].(map[string]any)["locale"] != "en" {
		t.Fatalf("PATCH /me should save locale: %v", body)
	}
	if _, body = doLang(h, call{method: "GET", path: "/api/v1/courses", cookie: ck}, ""); body["courses"].([]any)[0].(map[string]any)["title"] != "English" {
		t.Fatalf("new saved locale should apply: %v", body["courses"])
	}
}
