// Package middleware holds custom Echo middleware.
package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
)

const (
	// FrontendKeyHeader carries the shared key only our frontend's server knows.
	FrontendKeyHeader = "X-Frontend-Key"

	userEmailKey = "userEmail"
)

// TokenVerifier checks a session token and returns the user's email. It returns
// authservice.ErrInvalidToken for a token that isn't (or is no longer) a valid session.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (string, error)
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
			token, ok := BearerToken(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
			}
			email, err := verifier.VerifyToken(c.Request().Context(), token)
			if errors.Is(err, authservice.ErrInvalidToken) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired session")
			}
			// Anything else (e.g. the database is down) is a 500, not a 401: the frontend treats
			// a 401 as a dead session and deletes the cookie, so an outage would log you out
			if err != nil {
				return err
			}
			c.Set(userEmailKey, email)
			return next(c)
		}
	}
}

// BearerToken returns the token from an "Authorization: Bearer <token>" header, and false if
// there isn't one.
func BearerToken(c *echo.Context) (string, bool) {
	token, ok := strings.CutPrefix(c.Request().Header.Get(echo.HeaderAuthorization), "Bearer ")
	return token, ok && token != ""
}

// UserEmail returns the email RequireSession stored on the context.
func UserEmail(c *echo.Context) (string, error) {
	return echo.ContextGet[string](c, userEmailKey)
}
