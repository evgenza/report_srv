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
	app := fx.New(
		// Поставщики зависимостей
		fx.Provide(
			provideConfig,
			provideLogger,
			database.NewDatabase,
			storage.NewStorageFromConfig,
			service.NewReportServiceFromDB,
			server.NewServer,
		),

		// Хуки жизненного цикла
		fx.Invoke(registerLifecycleHooks),
	)

	// Запуск приложения с остановкой
	runWithGracefulShutdown(app)
}

// provideConfig загружает и предоставляет конфигурацию приложения
func provideConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// provideLogger создает и настраивает логгер на основе конфигурации
func provideLogger(cfg config.Config) *logrus.Logger {
	logger := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
		logger.WithError(err).Warn("Неверный уровень логирования, используется info")
	}
	logger.SetLevel(level)

	// Устанавливаем формат вывода
	switch cfg.Logging.Format {
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

	logger.WithField("config", cfg.String()).Info("Запуск сервиса отчетов")
	return logger
}

// registerLifecycleHooks настраивает хуки жизненного цикла приложения
func registerLifecycleHooks(
	srv server.HTTPServer,
	cfg config.Config,
	logger *logrus.Logger,
	lc fx.Lifecycle,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Запуск HTTP сервера")
			go func() {
				if err := srv.Start(cfg.Server.Address); err != nil {
					logger.WithError(err).Error("Не удалось запустить HTTP сервер")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Завершение работы HTTP сервера")
			return srv.Shutdown(ctx)
		},
	})
}

// runWithGracefulShutdown обрабатывает жизненный цикл приложения с обработкой сигналов
func runWithGracefulShutdown(app *fx.App) {
	// Создаем контексты
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем приложение с таймаутом
	startCtx, startCancel := context.WithTimeout(ctx, 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		logrus.WithError(err).Fatal("Не удалось запустить приложение")
	}

	// Ожидаем сигнал завершения
	<-quit
	logrus.Info("Получен сигнал завершения работы")

	// Грациозное завершение с таймаутом
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if err := app.Stop(stopCtx); err != nil {
		logrus.WithError(err).Error("Ошибка при завершении работы")
		os.Exit(1)
	}

	logrus.Info("Сервис отчетов остановлен корректно")
}
