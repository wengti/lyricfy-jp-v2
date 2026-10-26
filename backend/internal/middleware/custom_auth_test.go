package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

const testKey = "frontend-key-that-is-at-least-32-chars"

type fakeVerifier struct{}

func (fakeVerifier) VerifyToken(token string) (string, error) {
	if token == "valid-token" {
		return "me@example.com", nil
	}
	return "", errors.New("bad token")
}

// serve runs a request through mw in front of a handler that echoes the stored email.
func serve(mw echo.MiddlewareFunc, headers map[string]string) *httptest.ResponseRecorder {
	e := echo.New()
	e.GET("/", func(c *echo.Context) error {
		email, _ := UserEmail(c)
		return c.String(http.StatusOK, "ok:"+email)
	}, mw)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRequireFrontend(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{"correct key", map[string]string{FrontendKeyHeader: testKey}, http.StatusOK},
		{"missing key", nil, http.StatusUnauthorized},
		{"wrong key", map[string]string{FrontendKeyHeader: "wrong"}, http.StatusUnauthorized},
		{"key with extra characters", map[string]string{FrontendKeyHeader: testKey + "x"}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(RequireFrontend(testKey), tt.headers)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireSession(t *testing.T) {
	tests := []struct {
		name       string
		auth       string
		wantStatus int
		wantBody   string
	}{
		{"valid token stores the email", "Bearer valid-token", http.StatusOK, "ok:me@example.com"},
		{"missing header", "", http.StatusUnauthorized, ""},
		{"wrong scheme", "Basic valid-token", http.StatusUnauthorized, ""},
		{"empty token", "Bearer ", http.StatusUnauthorized, ""},
		{"invalid token", "Bearer nope", http.StatusUnauthorized, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(RequireSession(fakeVerifier{}), map[string]string{echo.HeaderAuthorization: tt.auth})
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
