package main

import (
	"context"
	"log"

	"github.com/lume/backend/internal/adapters/http"
	"github.com/lume/backend/internal/adapters/repositories"
	"github.com/lume/backend/internal/core/services"
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
func StartServer(lc fx.Lifecycle, app *mach.App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting Lume API on :3000")
			go func() {
				if err := app.Run(":3000"); err != nil {
					log.Fatalf("Server failed to start: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Lume API...")
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			// Adapters
			repositories.NewGCSDownloader,
			repositories.NewFirestoreWorkspaceRepository,
			http.NewWebhookHandler,
			http.NewWorkspaceHandler,

			// Core Services
			services.NewTofuParser,
			services.NewTofuService,

			// Server
			NewServer,
		),
		fx.Invoke(StartServer),
	)

	app.Run()
}
