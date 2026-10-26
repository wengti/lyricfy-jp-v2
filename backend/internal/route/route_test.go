package route

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/config"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/middleware"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

// These tests cover the wiring: which URLs exist and which middleware guards them.
// What each handler does with a request is tested in its own handler package.

const (
	testFrontendKey = "frontend-key-that-is-at-least-32-chars"
	testEmail       = "me@example.com"
	testPassword    = "correct-password"
	validLogin      = `{"email":"me@example.com","password":"correct-password"}`
)

// newTestServer registers every route through Register, exactly as main does.
func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Frontend: config.FrontendConfig{APIKey: testFrontendKey},
		Session:  config.SessionConfig{Secret: "session-secret-that-is-at-least-32-chars", TTL: time.Hour},
		User:     config.UserConfig{Email: testEmail, PasswordHash: string(hash)},
	}

	e := echo.New()
	e.HTTPErrorHandler = response.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	Register(e, cfg)
	return e
}

type requestOpts struct {
	body        string
	frontendKey string
	token       string
}

func do(t *testing.T, e *echo.Echo, method, path string, opts requestOpts) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(opts.body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if opts.frontendKey != "" {
		req.Header.Set(middleware.FrontendKeyHeader, opts.frontendKey)
	}
	if opts.token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+opts.token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return rec.Code, body
}

// loginToken logs in through the real route and returns the session token.
func loginToken(t *testing.T, e *echo.Echo) string {
	t.Helper()
	status, body := do(t, e, http.MethodPost, "/api/auth/login", requestOpts{body: validLogin, frontendKey: testFrontendKey})
	if status != http.StatusOK {
		t.Fatalf("login failed: %d %v", status, body)
	}
	return body["data"].(map[string]any)["token"].(string)
}

func TestRoutes_AccessRules(t *testing.T) {
	e := newTestServer(t)
	token := loginToken(t, e)

	tests := []struct {
		name       string
		method     string
		path       string
		opts       requestOpts
		wantStatus int
	}{
		// Login is public apart from the frontend key
		{"login with frontend key", http.MethodPost, "/api/auth/login", requestOpts{body: validLogin, frontendKey: testFrontendKey}, http.StatusOK},
		{"login without frontend key", http.MethodPost, "/api/auth/login", requestOpts{body: validLogin}, http.StatusUnauthorized},
		{"login with wrong frontend key", http.MethodPost, "/api/auth/login", requestOpts{body: validLogin, frontendKey: "wrong"}, http.StatusUnauthorized},

		// /me needs the frontend key and a session
		{"me with key and session", http.MethodGet, "/api/auth/me", requestOpts{frontendKey: testFrontendKey, token: token}, http.StatusOK},
		{"me without session", http.MethodGet, "/api/auth/me", requestOpts{frontendKey: testFrontendKey}, http.StatusUnauthorized},
		{"me with invalid session", http.MethodGet, "/api/auth/me", requestOpts{frontendKey: testFrontendKey, token: "garbage"}, http.StatusUnauthorized},
		{"me with session but no frontend key", http.MethodGet, "/api/auth/me", requestOpts{token: token}, http.StatusUnauthorized},

		// Unknown API paths don't reveal whether they exist to logged-out callers
		{"unknown path without session", http.MethodGet, "/api/nope", requestOpts{frontendKey: testFrontendKey}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := do(t, e, tt.method, tt.path, tt.opts)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %v)", status, tt.wantStatus, body)
			}
		})
	}
}

func TestRoutes_LoginThenMe(t *testing.T) {
	e := newTestServer(t)
	token := loginToken(t, e)

	status, body := do(t, e, http.MethodGet, "/api/auth/me", requestOpts{frontendKey: testFrontendKey, token: token})

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
	if email := body["data"].(map[string]any)["email"]; email != testEmail {
		t.Errorf("email = %v, want %q", email, testEmail)
	}
}
