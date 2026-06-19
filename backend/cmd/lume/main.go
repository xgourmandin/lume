package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lume/backend/internal/adapters/http"
	"github.com/lume/backend/internal/adapters/repositories"
	"github.com/lume/backend/internal/core/services"
	"github.com/lume/backend/internal/platform/logging"
	"github.com/mrshabel/mach"
	"go.uber.org/fx"
)

// NewServer initializes the Mach application.
func NewServer(webhookHandler *http.WebhookHandler, workspaceHandler *http.WorkspaceHandler) *mach.App {
	app := mach.New()

	app.Use(mach.Logger())
	app.Use(mach.Recovery())

	app.Use(mach.CORS([]string{"*"}))

	// Basic health check
	app.GET("/health", func(c *mach.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	// Register adapters
	webhookHandler.Register(app)
	workspaceHandler.Register(app)

	return app
}

// StartServer hook to manage the server lifecycle via FX.
func StartServer(lc fx.Lifecycle, app *mach.App, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting Lume API", "addr", ":3000")
			go func() {
				if err := app.Run(":3000"); err != nil {
					logger.Error("server failed to start", "error", err)
					os.Exit(1)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping Lume API")
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			// Platform
			logging.NewLogger,

			// Adapters
			repositories.NewGCSDownloader,
			repositories.NewFirestoreWorkspaceRepository,
			http.NewWebhookHandler,
			http.NewWorkspaceHandler,

			// Core Services
			services.NewTofuParser,
			services.NewTofuPlanParser,
			services.NewTofuService,

			// Server
			NewServer,
		),
		fx.Invoke(StartServer),
	)

	app.Run()
}
