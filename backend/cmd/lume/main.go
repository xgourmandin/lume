package main

import (
	"context"
	"log"

	"github.com/mrshabel/mach"
	"go.uber.org/fx"
)

// NewServer initializes the Mach application.
func NewServer() *mach.App {
	app := mach.New()

	app.Use(mach.Logger())
	app.Use(mach.Recovery())

	// Basic health check route
	app.GET("/health", func(c *mach.Context) {
		c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	return app
}

// StartServer hook to manage the server lifecycle via FX.
func StartServer(lc fx.Lifecycle, app *mach.App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Starting Mach server on :3000")
			go func() {
				if err := app.Run(":3000"); err != nil {
					log.Fatalf("Server failed to start: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Stopping Mach server...")
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			NewServer,
			// Future providers: Repositories, Services, Adapters...
		),
		fx.Invoke(StartServer),
	)

	app.Run()
}
