package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"report_srv/internal/config"
	"report_srv/internal/database"
	"report_srv/internal/server"
	"report_srv/internal/service"
	"report_srv/internal/storage"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Настраиваем логирование
	logger := setupLogger(cfg.Logging)
	logger.WithField("config", cfg.String()).Info("Starting report service")

	app := fx.New(
		// Предоставляем зависимости
		fx.Provide(
			func() config.Config { return cfg },
			func() *logrus.Logger { return logger },
			database.NewDatabase,
			storage.NewStorageFromConfig,
			service.NewReportService,
			server.NewServer,
		),

		// Запускаем приложение
		fx.Invoke(func(
			srv *server.Server,
			logger *logrus.Logger,
			lc fx.Lifecycle,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info("Starting HTTP server")
					go func() {
						if err := srv.Start(cfg.Server.Address); err != nil {
							logger.WithError(err).Error("HTTP server stopped")
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					logger.Info("Shutting down HTTP server")
					return srv.Shutdown(ctx)
				},
			})
		}),
	)

	// Настраиваем graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Слушаем сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем приложение в отдельной горутине
	go func() {
		if err := app.Start(ctx); err != nil {
			logger.WithError(err).Fatal("Failed to start application")
		}
	}()

	// Ждем сигнал завершения
	<-quit
	logger.Info("Received shutdown signal")

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Останавливаем приложение
	if err := app.Stop(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}

	logger.Info("Report service stopped")
}

// setupLogger настраивает логгер согласно конфигурации
func setupLogger(logCfg config.Logging) *logrus.Logger {
	logger := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(logCfg.Level)
	if err != nil {
		level = logrus.InfoLevel
		logger.WithError(err).Warn("Invalid log level, using info")
	}
	logger.SetLevel(level)

	// Устанавливаем формат вывода
	switch logCfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return logger
}
