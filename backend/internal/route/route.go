// Package route wires URLs to handlers.
//
// Register builds what several features share (middleware groups, shared services) and
// then calls one register function per feature. Each feature builds its own repository,
// service and handler inside its register function, unless they're shared.
package route

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/config"
	authhandler "github.com/wengti/lyricfy-jp-v2/backend/internal/handler/auth"
	dicthandler "github.com/wengti/lyricfy-jp-v2/backend/internal/handler/dictionary"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/middleware"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/repository"
	authservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/auth"
	dictservice "github.com/wengti/lyricfy-jp-v2/backend/internal/service/dictionary"
)

// groups are the two access levels every feature can mount routes on.
type groups struct {
	public    *echo.Group // frontend key only, mounted at /api
	protected *echo.Group // frontend key + logged in, mounted at /api
}

// Register mounts every route under /api. Features build their repositories from db.
func Register(e *echo.Echo, cfg *config.Config, db *pgxpool.Pool) {
	// Shared: the session middleware on every protected route needs the auth service
	users := repository.NewConfigUserRepository(cfg.User.Email, cfg.User.PasswordHash)
	sessions := repository.NewPostgresSessionRepository(db)
	authSvc := authservice.NewService(users, sessions, cfg.Session.TTL)

	api := e.Group("/api", middleware.RequireFrontend(cfg.Frontend.APIKey))
	g := groups{
		public:    api,
		protected: api.Group("", middleware.RequireSession(authSvc)),
	}

	registerAuth(g, authSvc)
	registerDictionary(g, db)
}

// registerDictionary mounts /api/dictionary, for the logged-in user only.
func registerDictionary(g groups, db *pgxpool.Pool) {
	svc := dictservice.NewService(repository.NewPostgresDictionaryRepository(db))
	h := dicthandler.NewHandler(svc)

	g.protected.GET("/dictionary", h.List)
}

// registerAuth mounts /api/auth/*. The auth service is passed in because it's shared.
func registerAuth(g groups, authSvc *authservice.Service) {
	h := authhandler.NewHandler(authSvc)

	public := g.public.Group("/auth")
	public.POST("/login", h.Login)
	// Public so an expired session can still log out; it only ends the session it's given
	public.POST("/logout", h.Logout)

	g.protected.Group("/auth").GET("/me", h.Me)
}
