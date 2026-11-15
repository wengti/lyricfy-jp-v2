// Command server runs the Lyricfy JP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"github.com/wengti/lyricfy-jp-v2/backend/internal/config"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/database"
	"github.com/wengti/lyricfy-jp-v2/backend/internal/route"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// On Cloud Run the config comes from Parameter Manager; fail rather than hang if it's unreachable
	loadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cfg, err := config.LoadFromEnv(loadCtx)
	cancel()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect and bring the schema up to date before serving. Neon may be waking from suspend,
	// so allow it time, but fail rather than hang.
	dbCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	db, err := database.Open(dbCtx, cfg.Database.URL)
	if err != nil {
		cancel()
		logger.Error("failed to connect to the database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	applied, err := database.Migrate(dbCtx, db)
	cancel()
	if err != nil {
		logger.Error("failed to migrate the database", "error", err)
		os.Exit(1)
	}
	logger.Info("database ready", "migrations_applied", applied)

	e := echo.New()
	e.Logger = logger
	e.HTTPErrorHandler = response.ErrorHandler(logger)
	e.Use(echomw.Recover())
	e.Use(echomw.RequestLogger())
	e.Use(echomw.BodyLimit(1 << 20)) // 1 MiB

	route.Register(e, cfg, db)

	// Stop gracefully on Ctrl+C locally, or SIGTERM from Cloud Run
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{Address: ":" + cfg.Server.Port, GracefulTimeout: 10 * time.Second}
	if err := sc.Start(ctx, e); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
