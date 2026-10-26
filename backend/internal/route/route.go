// Package route wires URLs to handlers.
//
// Register builds what several features share (middleware groups, shared services) and
// then calls one register function per feature. Each feature builds its own repository,
// service and handler inside its register function, unless they're shared.
package route

import (
	"github.com/labstack/echo/v5"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/config"
	authhandler "github.com/wengti/lyricfy-jp-v2/backend/internal/handler/auth"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/middleware"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
)

// groups are the two access levels every feature can mount routes on.
type groups struct {
	public    *echo.Group // frontend key only, mounted at /api
	protected *echo.Group // frontend key + logged in, mounted at /api
}

// Register mounts every route under /api.
func Register(e *echo.Echo, cfg *config.Config) {
	// Shared: the session middleware on every protected route needs the auth service
	users := repository.NewConfigUserRepository(cfg.User.Email, cfg.User.PasswordHash)
	authSvc := authservice.NewService(users, cfg.Session.Secret, cfg.Session.TTL)

	api := e.Group("/api", middleware.RequireFrontend(cfg.Frontend.APIKey))
	g := groups{
		public:    api,
		protected: api.Group("", middleware.RequireSession(authSvc)),
	}

	registerAuth(g, authSvc)
}

// registerAuth mounts /api/auth/*. The auth service is passed in because it's shared.
func registerAuth(g groups, authSvc *authservice.Service) {
	h := authhandler.NewHandler(authSvc)

	g.public.Group("/auth").POST("/login", h.Login)
	g.protected.Group("/auth").GET("/me", h.Me)
}
