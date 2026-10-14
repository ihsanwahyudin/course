package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"fluentpath/api/internal/apperr"
	"fluentpath/api/internal/storage"
)

const SessionTTL = 30 * 24 * time.Hour

type Service struct {
	Store *storage.Store
}

type Registration struct {
	Email       string
	Password    string
	DisplayName string
}

func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validate(r Registration) error {
	addr, err := mail.ParseAddress(r.Email)
	if err != nil || addr.Address != r.Email || !strings.Contains(r.Email[strings.LastIndex(r.Email, "@"):], ".") || len(r.Email) > 254 {
		return apperr.New(400, "invalid_email", false, "Enter a valid email address.")
	}
	if n := utf8.RuneCountInString(r.Password); n < 8 || n > 128 {
		return apperr.New(400, "invalid_password", false, "Password must be 8–128 characters.")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(r.DisplayName)); n < 1 || n > 60 {
		return apperr.New(400, "invalid_name", false, "Name must be 1–60 characters.")
	}
	return nil
}

// Register creates an account and logs it in. Returns the new session token.
func (s *Service) Register(ctx context.Context, r Registration, userAgent string) (storage.User, string, error) {
	r.Email = NormalizeEmail(r.Email)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if err := validate(r); err != nil {
		return storage.User{}, "", err
	}
	hash, err := HashPassword(r.Password)
	if err != nil {
		return storage.User{}, "", err
	}
	u, err := storage.CreateUser(ctx, s.Store.Pool, r.Email, hash, r.DisplayName)
	if errors.Is(err, storage.ErrConflict) {
		return u, "", apperr.New(409, "email_taken", false, "An account with this email already exists. Try logging in.")
	}
	if err != nil {
		return u, "", err
	}
	token, err := s.newSession(ctx, u.ID, userAgent)
	return u, token, err
}

var errBadLogin = apperr.New(401, "invalid_credentials", false, "Wrong email or password.")

// Login checks credentials and starts a session. Unknown emails and wrong
// passwords produce the same error and take similar time.
func (s *Service) Login(ctx context.Context, email, password, userAgent string) (storage.User, string, error) {
	u, err := storage.GetUserByEmail(ctx, s.Store.Pool, NormalizeEmail(email))
	if errors.Is(err, storage.ErrNotFound) || (err == nil && u.PasswordHash == "") {
		BurnTime(password)
		return storage.User{}, "", errBadLogin
	}
	if err != nil {
		return u, "", err
	}
	if !VerifyPassword(u.PasswordHash, password) {
		return storage.User{}, "", errBadLogin
	}
	token, err := s.newSession(ctx, u.ID, userAgent)
	return u, token, err
}

func (s *Service) newSession(ctx context.Context, userID, userAgent string) (string, error) {
	token, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	return token, storage.CreateAuthSession(ctx, s.Store.Pool, userID, hash, userAgent, time.Now().Add(SessionTTL))
}

// Authenticate resolves a bearer token to its user (sliding expiry).
func (s *Service) Authenticate(ctx context.Context, token string) (storage.User, error) {
	if token == "" || len(token) > 200 {
		return storage.User{}, apperr.Unauthorized
	}
	u, err := storage.UserForToken(ctx, s.Store.Pool, HashToken(token), SessionTTL)
	if errors.Is(err, storage.ErrNotFound) {
		return u, apperr.Unauthorized
	}
	return u, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return storage.DeleteAuthSession(ctx, s.Store.Pool, HashToken(token))
}
