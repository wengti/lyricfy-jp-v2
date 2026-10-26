package auth

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

	"github.com/wengti/lyricfy-jp-v2/backend/internal/middleware"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

const (
	testEmail    = "me@example.com"
	testPassword = "correct-password"
)

// These tests cover the handlers on their own. Which URL they're mounted on and which
// middleware guards them is tested in the route package.

func newHandler(t *testing.T) *Handler {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	svc := authservice.NewService(
		repository.NewConfigUserRepository(testEmail, string(hash)),
		"session-secret-that-is-at-least-32-chars",
		time.Hour,
	)
	return NewHandler(svc)
}

// serve sends one request to h and returns the status and decoded JSON body.
func serve(t *testing.T, method string, h echo.HandlerFunc, body string) (int, map[string]any) {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = response.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.Add(method, "/", h)

	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return rec.Code, decoded
}

func TestLogin_ReturnsSessionToken(t *testing.T) {
	h := newHandler(t)

	status, body := serve(t, http.MethodPost, h.Login, `{"email":"me@example.com","password":"correct-password"}`)

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if token, _ := data["token"].(string); token == "" {
		t.Errorf("expected a token, got %v", body)
	}
	if _, err := time.Parse(time.RFC3339, data["expiresAt"].(string)); err != nil {
		t.Errorf("expiresAt is not a timestamp: %v", data["expiresAt"])
	}
}

func TestLogin_Errors(t *testing.T) {
	h := newHandler(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{"wrong password", `{"email":"me@example.com","password":"nope"}`, http.StatusUnauthorized, "invalid email or password"},
		{"unknown email gets the same message", `{"email":"x@example.com","password":"correct-password"}`, http.StatusUnauthorized, "invalid email or password"},
		{"missing password", `{"email":"me@example.com"}`, http.StatusBadRequest, "email and password are required"},
		{"blank email", `{"email":"  ","password":"correct-password"}`, http.StatusBadRequest, "email and password are required"},
		{"malformed JSON", `{"email":`, http.StatusBadRequest, "invalid request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := serve(t, http.MethodPost, h.Login, tt.body)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %v)", status, tt.wantStatus, body)
			}
			if body["error"] != tt.wantError {
				t.Errorf("error = %v, want %q", body["error"], tt.wantError)
			}
		})
	}
}

// stubVerifier accepts any token as the test user, so Me can be tested without logging in.
type stubVerifier struct{}

func (stubVerifier) VerifyToken(string) (string, error) { return testEmail, nil }

func TestMe_ReturnsLoggedInEmail(t *testing.T) {
	h := newHandler(t)

	// Me reads the email that the session middleware stores on the request
	e := echo.New()
	e.GET("/", h.Me, middleware.RequireSession(stubVerifier{}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer any-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"data":{"email":"me@example.com"}}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}
