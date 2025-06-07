package database

import (
	"context"
	"fmt"
	"time"

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

const (
	// Значения по умолчанию для пула соединений
	defaultMaxIdleConns    = 10
	defaultMaxOpenConns    = 100
	defaultConnMaxLifetime = time.Hour
)

// Database интерфейс для работы с базой данных
type Database interface {
	DB() *gorm.DB
	Close() error
	Ping(ctx context.Context) error
	RunMigrations(ctx context.Context) error
}

// ConnectionConfig настройки пула соединений
type ConnectionConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// DatabaseManager управляет подключением к базе данных
type DatabaseManager struct {
	db     *gorm.DB
	logger *logrus.Logger
	config config.Config
}

// DriverFactory фабрика для создания драйверов БД
type DriverFactory interface {
	CreateDialector(dsn string) gorm.Dialector
	SupportsDriver(driver string) bool
}

// PostgresDriverFactory фабрика для PostgreSQL
type PostgresDriverFactory struct{}

func (f *PostgresDriverFactory) CreateDialector(dsn string) gorm.Dialector {
	return postgres.Open(dsn)
}

func (f *PostgresDriverFactory) SupportsDriver(driver string) bool {
	return driver == "postgres"
}

// SQLiteDriverFactory фабрика для SQLite
type SQLiteDriverFactory struct{}

func (f *SQLiteDriverFactory) CreateDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}

func (f *SQLiteDriverFactory) SupportsDriver(driver string) bool {
	return driver == "sqlite"
}

// Migrator интерфейс для выполнения миграций
type Migrator interface {
	Migrate(ctx context.Context, db *gorm.DB) error
}

// AutoMigrator выполняет автоматические миграции GORM
type AutoMigrator struct {
	logger *logrus.Logger
	models []interface{}
}

// NewAutoMigrator создает новый AutoMigrator
func NewAutoMigrator(logger *logrus.Logger) *AutoMigrator {
	return &AutoMigrator{
		logger: logger,
		models: []interface{}{
			&models.Report{},
			// Здесь можно добавить другие модели
		},
	}
}

// Migrate выполняет миграции
func (m *AutoMigrator) Migrate(ctx context.Context, db *gorm.DB) error {
	m.logger.Info("Запуск миграций базы данных")

	for _, model := range m.models {
		if err := db.WithContext(ctx).AutoMigrate(model); err != nil {
			return fmt.Errorf("ошибка миграции модели %T: %w", model, err)
		}
	}

	m.logger.Info("Миграции базы данных выполнены успешно")
	return nil
}

// DatabaseBuilder строитель для конфигурации базы данных
type DatabaseBuilder struct {
	config           config.Config
	logger           *logrus.Logger
	connectionConfig ConnectionConfig
	driverFactories  []DriverFactory
	migrator         Migrator
}

// NewDatabaseBuilder создает новый DatabaseBuilder
func NewDatabaseBuilder(cfg config.Config, logger *logrus.Logger) *DatabaseBuilder {
	return &DatabaseBuilder{
		config: cfg,
		logger: logger,
		connectionConfig: ConnectionConfig{
			MaxIdleConns:    defaultMaxIdleConns,
			MaxOpenConns:    defaultMaxOpenConns,
			ConnMaxLifetime: defaultConnMaxLifetime,
		},
		driverFactories: []DriverFactory{
			&PostgresDriverFactory{},
			&SQLiteDriverFactory{},
		},
		migrator: NewAutoMigrator(logger),
	}
}

// WithConnectionConfig устанавливает настройки пула соединений
func (b *DatabaseBuilder) WithConnectionConfig(config ConnectionConfig) *DatabaseBuilder {
	b.connectionConfig = config
	return b
}

// WithMigrator устанавливает кастомный мигратор
func (b *DatabaseBuilder) WithMigrator(migrator Migrator) *DatabaseBuilder {
	b.migrator = migrator
	return b
}

// WithDriverFactory добавляет фабрику драйверов
func (b *DatabaseBuilder) WithDriverFactory(factory DriverFactory) *DatabaseBuilder {
	b.driverFactories = append(b.driverFactories, factory)
	return b
}

// Build создает и настраивает подключение к базе данных
func (b *DatabaseBuilder) Build(ctx context.Context) (Database, error) {
	gormConfig := b.createGormConfig()

	dialector, err := b.createDialector()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания диалектора: %w", err)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	manager := &DatabaseManager{
		db:     db,
		logger: b.logger,
		config: b.config,
	}

	if err := manager.configureConnectionPool(b.connectionConfig); err != nil {
		return nil, fmt.Errorf("ошибка настройки пула соединений: %w", err)
	}

	if err := manager.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ошибка проверки подключения: %w", err)
	}

	b.logger.WithField("driver", b.config.DB.Driver).Info("База данных подключена успешно")
	return manager, nil
}

// createGormConfig создает конфигурацию GORM
func (b *DatabaseBuilder) createGormConfig() *gorm.Config {
	var logLevel logger.LogLevel
	if b.config.Server.Debug {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	return &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// createDialector создает диалектор для указанного драйвера
func (b *DatabaseBuilder) createDialector() (gorm.Dialector, error) {
	for _, factory := range b.driverFactories {
		if factory.SupportsDriver(b.config.DB.Driver) {
			return factory.CreateDialector(b.config.DB.DSN), nil
		}
	}
	return nil, fmt.Errorf("неподдерживаемый драйвер базы данных: %s", b.config.DB.Driver)
}

// DB возвращает экземпляр GORM DB
func (dm *DatabaseManager) DB() *gorm.DB {
	return dm.db
}

// Close закрывает подключение к базе данных
func (dm *DatabaseManager) Close() error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения SQL DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия соединения с БД: %w", err)
	}

	dm.logger.Info("Соединение с базой данных закрыто")
	return nil
}

// Ping проверяет доступность базы данных
func (dm *DatabaseManager) Ping(ctx context.Context) error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения SQL DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ошибка проверки соединения с БД: %w", err)
	}

	return nil
}

// RunMigrations запускает миграции базы данных
func (dm *DatabaseManager) RunMigrations(ctx context.Context) error {
	return dm.runMigrations(ctx, NewAutoMigrator(dm.logger))
}

// runMigrations внутренний метод для запуска миграций
func (dm *DatabaseManager) runMigrations(ctx context.Context, migrator Migrator) error {
	return migrator.Migrate(ctx, dm.db)
}

// configureConnectionPool настраивает пул соединений
func (dm *DatabaseManager) configureConnectionPool(config ConnectionConfig) error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения SQL DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	dm.logger.WithFields(logrus.Fields{
		"max_idle_conns":    config.MaxIdleConns,
		"max_open_conns":    config.MaxOpenConns,
		"conn_max_lifetime": config.ConnMaxLifetime,
	}).Info("Пул соединений настроен")

	return nil
}

// NewDatabase создает новое подключение к базе данных (обратная совместимость)
func NewDatabase(cfg config.Config, log *logrus.Logger) (*gorm.DB, error) {
	ctx := context.Background()

	database, err := NewDatabaseBuilder(cfg, log).Build(ctx)
	if err != nil {
		return nil, err
	}

	return database.DB(), nil
}

// NewDatabaseWithMigrations создает подключение и выполняет миграции
func NewDatabaseWithMigrations(cfg config.Config, log *logrus.Logger) (Database, error) {
	ctx := context.Background()

	database, err := NewDatabaseBuilder(cfg, log).Build(ctx)
	if err != nil {
		return nil, err
	}

	if err := database.RunMigrations(ctx); err != nil {
		return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
	}

	return database, nil
}
