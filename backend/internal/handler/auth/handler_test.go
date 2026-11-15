package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository/repotest"
	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

const (
	testEmail    = "me@example.com"
	testPassword = "correct-password"
)

// These tests cover the handlers on their own. Which URL they're mounted on and which
// middleware guards them is tested in the route package.

func newHandler(t *testing.T) (*Handler, *authservice.Service, *repotest.SessionRepository) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	sessions := repotest.NewSessionRepository()
	svc := authservice.NewService(repository.NewConfigUserRepository(testEmail, string(hash)), sessions, time.Hour)
	return NewHandler(svc), svc, sessions
}

// serve sends one request to h and returns the status and decoded JSON body.
func serve(t *testing.T, method string, h echo.HandlerFunc, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = response.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.Add(method, "/", h)

	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return rec.Code, decoded
}

func bearer(token string) map[string]string {
	return map[string]string{echo.HeaderAuthorization: "Bearer " + token}
}

func TestLogin_ReturnsSessionToken(t *testing.T) {
	h, _, _ := newHandler(t)

	status, body := serve(t, http.MethodPost, h.Login, `{"email":"me@example.com","password":"correct-password"}`, nil)

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
	h, _, _ := newHandler(t)

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
			status, body := serve(t, http.MethodPost, h.Login, tt.body, nil)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %v)", status, tt.wantStatus, body)
			}
			if body["error"] != tt.wantError {
				t.Errorf("error = %v, want %q", body["error"], tt.wantError)
			}
		})
	}
}

func TestLogin_DatabaseErrorIsHidden(t *testing.T) {
	h, _, sessions := newHandler(t)
	sessions.Err = errors.New("connection refused to db.internal")

	status, body := serve(t, http.MethodPost, h.Login, `{"email":"me@example.com","password":"correct-password"}`, nil)

	if status != http.StatusInternalServerError || body["error"] != "Internal Server Error" {
		t.Fatalf("got %d %v, want a generic 500", status, body)
	}
}

func TestLogout_EndsTheSession(t *testing.T) {
	h, svc, _ := newHandler(t)
	session, err := svc.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	status, body := serve(t, http.MethodPost, h.Logout, "", bearer(session.Token))

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
	if _, err := svc.VerifyToken(context.Background(), session.Token); !errors.Is(err, authservice.ErrInvalidToken) {
		t.Errorf("token still valid after logout: err = %v", err)
	}
}

func TestLogout_UnknownSessionSucceeds(t *testing.T) {
	h, _, _ := newHandler(t)

	status, body := serve(t, http.MethodPost, h.Logout, "", bearer("not-a-session"))

	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %v", status, body)
	}
}

func TestLogout_Errors(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		h, _, _ := newHandler(t)
		status, body := serve(t, http.MethodPost, h.Logout, "", nil)
		if status != http.StatusUnauthorized || body["error"] != "not logged in" {
			t.Fatalf("got %d %v", status, body)
		}
	})
	t.Run("database error", func(t *testing.T) {
		h, svc, sessions := newHandler(t)
		session, err := svc.Login(context.Background(), testEmail, testPassword)
		if err != nil {
			t.Fatal(err)
		}
		sessions.Err = errors.New("connection refused")

		status, _ := serve(t, http.MethodPost, h.Logout, "", bearer(session.Token))
		if status != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", status)
		}
	})
}

// stubVerifier accepts any token as the test user, so Me can be tested without logging in.
type stubVerifier struct{}

func (stubVerifier) VerifyToken(context.Context, string) (string, error) { return testEmail, nil }

func TestMe_ReturnsLoggedInEmail(t *testing.T) {
	h, _, _ := newHandler(t)

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
