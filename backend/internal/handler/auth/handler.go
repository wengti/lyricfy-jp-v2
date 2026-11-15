// Package auth holds the HTTP handlers for /api/auth/*: they bind requests,
// call the auth service and write responses. URLs are mapped in the route package.
package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/middleware"
	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

type Handler struct {
	authSvc *authservice.Service
}

func NewHandler(authSvc *authservice.Service) *Handler {
	return &Handler{authSvc: authSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/auth/login.
func (h *Handler) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	session, err := h.authSvc.Login(c.Request().Context(), req.Email, req.Password)
	if errors.Is(err, authservice.ErrInvalidCredentials) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
	}
	if err != nil {
		return err
	}
	return response.OK(c, session)
}

// Logout handles POST /api/auth/logout: it ends the session in the Authorization header, so
// the token stops working even before it expires. Ending an expired or unknown session also
// succeeds, so the frontend can always go on to clear its cookie.
func (h *Handler) Logout(c *echo.Context) error {
	token, ok := middleware.BearerToken(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}
	if err := h.authSvc.Logout(c.Request().Context(), token); err != nil {
		return err
	}
	return response.OK(c, nil)
}

type meResponse struct {
	Email string `json:"email"`
}

// Me handles GET /api/auth/me and returns the logged-in user.
func (h *Handler) Me(c *echo.Context) error {
	email, err := middleware.UserEmail(c)
	if err != nil {
		return err
	}
	return response.OK(c, meResponse{Email: email})
}
