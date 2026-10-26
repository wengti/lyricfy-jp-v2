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
	"github.com/wengti/lyricfy-jp-v2/backend/internal/route"
	"github.com/wengti/lyricfy-jp-v2/backend/pkg/response"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	e := echo.New()
	e.Logger = logger
	e.HTTPErrorHandler = response.ErrorHandler(logger)
	e.Use(echomw.Recover())
	e.Use(echomw.RequestLogger())
	e.Use(echomw.BodyLimit(1 << 20)) // 1 MiB

	route.Register(e, cfg)

	// Stop gracefully on Ctrl+C locally, or SIGTERM from Cloud Run
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{Address: ":" + cfg.Server.Port, GracefulTimeout: 10 * time.Second}
	if err := sc.Start(ctx, e); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
