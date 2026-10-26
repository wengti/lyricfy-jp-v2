// Package middleware holds custom Echo middleware.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

const (
	// FrontendKeyHeader carries the shared key only our frontend's server knows.
	FrontendKeyHeader = "X-Frontend-Key"

	userEmailKey = "userEmail"
)

// TokenVerifier checks a session token and returns the user's email.
type TokenVerifier interface {
	VerifyToken(token string) (string, error)
}

// RequireFrontend rejects requests that don't carry the frontend's shared key.
func RequireFrontend(apiKey string) echo.MiddlewareFunc {
	expected := []byte(apiKey)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			got := []byte(c.Request().Header.Get(FrontendKeyHeader))
			// Constant-time comparison so response timing doesn't leak the key
			if subtle.ConstantTimeCompare(got, expected) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}

// RequireSession rejects requests without a valid "Authorization: Bearer <token>" header
// and stores the logged-in user's email on the context for handlers.
func RequireSession(verifier TokenVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, ok := strings.CutPrefix(c.Request().Header.Get(echo.HeaderAuthorization), "Bearer ")
			if !ok || token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
			}
			email, err := verifier.VerifyToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired session")
			}
			c.Set(userEmailKey, email)
			return next(c)
		}
	}
}

// UserEmail returns the email RequireSession stored on the context.
func UserEmail(c *echo.Context) (string, error) {
	return echo.ContextGet[string](c, userEmailKey)
}
