package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"report_srv/internal/database"
	"report_srv/internal/server"
	"report_srv/internal/service"
	"report_srv/internal/storage"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// Provide dependencies
		fx.Provide(
			// Database
			func() *database.Config {
				return &database.Config{
					Driver: os.Getenv("APP_DATABASE_DRIVER"),
					DSN:    os.Getenv("APP_DATABASE_DSN"),
					Debug:  os.Getenv("APP_SERVER_DEBUG") == "true",
				}
			},
			database.NewDatabase,

			// Storage
			func() *storage.S3Config {
				return &storage.S3Config{
					Region:    os.Getenv("APP_STORAGE_S3_REGION"),
					Bucket:    os.Getenv("APP_STORAGE_S3_BUCKET"),
					Endpoint:  os.Getenv("APP_STORAGE_S3_ENDPOINT"),
					AccessKey: os.Getenv("APP_STORAGE_S3_ACCESS_KEY"),
					SecretKey: os.Getenv("APP_STORAGE_S3_SECRET_KEY"),
				}
			},
			storage.NewS3Storage,

			// Server
			func() *server.Config {
				return &server.Config{
					Address: os.Getenv("APP_SERVER_ADDRESS"),
					Debug:   os.Getenv("APP_SERVER_DEBUG") == "true",
				}
			},
			server.NewServer,

			// Service
			service.NewReportService,
		),

		// Invoke startup
		fx.Invoke(func(
			db *database.Config,
			srv *server.Server,
			lc fx.Lifecycle,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Start the server
					go func() {
						if err := srv.Start(db.DSN); err != nil {
							log.Fatalf("Failed to start server: %v", err)
						}
					}()

					// Wait for interrupt signal
					quit := make(chan os.Signal, 1)
					signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
					<-quit

					return nil
				},
				OnStop: func(ctx context.Context) error {
					return nil
				},
			})
		}),
	)

	app.Run()
}
