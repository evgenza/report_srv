package database

import (
	"fmt"

	"report_srv/internal/config"
	"report_srv/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Импорт драйверов БД
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// NewDatabase creates a new database connection using GORM
func NewDatabase(cfg config.Config, log *logrus.Logger) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if cfg.Server.Debug {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	var db *gorm.DB
	var err error

	switch cfg.DB.Driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DB.DSN), gormConfig)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DB.DSN), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DB.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run auto migrations
	if err := runMigrations(db, log); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.WithField("driver", cfg.DB.Driver).Info("Database connected successfully")
	return db, nil
}

// runMigrations runs database migrations
func runMigrations(db *gorm.DB, log *logrus.Logger) error {
	log.Info("Running database migrations")

	// Migrate the models
	if err := db.AutoMigrate(&models.Report{}); err != nil {
		return fmt.Errorf("failed to migrate reports table: %w", err)
	}

	log.Info("Database migrations completed successfully")
	return nil
}
