package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const (
	// Значения по умолчанию для сервера
	defaultServerAddress = ":8080"
	defaultServerDebug   = true

	// Значения по умолчанию для базы данных
	defaultDBDriver = "postgres"
	defaultDBDSN    = "postgres://user:pass@localhost:5432/reports?sslmode=disable"

	// Значения по умолчанию для хранилища
	defaultStorageType     = "local"
	defaultStorageBasePath = "./templates"
	defaultS3Region        = "us-east-1"
	defaultS3Bucket        = "report-srv-bucket"

	// Значения по умолчанию для логирования
	defaultLogLevel  = "debug"
	defaultLogFormat = "text"

	// Префикс для переменных окружения
	envPrefix = "APP"
)

// Server содержит настройки HTTP-сервера
type Server struct {
	Address string `mapstructure:"address"`
	Debug   bool   `mapstructure:"debug"`
}

// DB содержит параметры подключения к БД
type DB struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// Storage описывает настройки хранилища файлов
type Storage struct {
	Type     string `mapstructure:"type"`
	BasePath string `mapstructure:"basepath"`
	S3       S3     `mapstructure:"s3"`
}

// S3 содержит настройки для S3-совместимого хранилища
type S3 struct {
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// Logging содержит настройки логирования
type Logging struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Config объединяет все разделы конфигурации
type Config struct {
	Server  Server  `mapstructure:"server"`
	DB      DB      `mapstructure:"database"`
	Storage Storage `mapstructure:"storage"`
	Logging Logging `mapstructure:"logging"`
}

// ConfigLoader интерфейс для загрузки конфигурации
type ConfigLoader interface {
	Load() (Config, error)
}

// ViperConfigLoader реализация загрузчика конфигурации на основе Viper
type ViperConfigLoader struct {
	configPaths []string
}

// NewConfigLoader создает новый загрузчик конфигурации
func NewConfigLoader(configPaths ...string) ConfigLoader {
	if len(configPaths) == 0 {
		configPaths = []string{".", "./config", "/etc/report-service"}
	}
	return &ViperConfigLoader{configPaths: configPaths}
}

// Load читает конфигурацию из файла и окружения с помощью viper
func Load() (Config, error) {
	loader := NewConfigLoader()
	return loader.Load()
}

// Load реализует загрузку конфигурации
func (l *ViperConfigLoader) Load() (Config, error) {
	if err := l.setupViper(); err != nil {
		return Config{}, fmt.Errorf("ошибка настройки viper: %w", err)
	}

	if err := l.readConfig(); err != nil {
		return Config{}, fmt.Errorf("ошибка чтения конфигурации: %w", err)
	}

	cfg, err := l.unmarshalConfig()
	if err != nil {
		return Config{}, fmt.Errorf("ошибка разбора конфигурации: %w", err)
	}

	if err := l.validateConfig(cfg); err != nil {
		return Config{}, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return cfg, nil
}

// setupViper настраивает viper с путями, переменными окружения и значениями по умолчанию
func (l *ViperConfigLoader) setupViper() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Добавляем пути для поиска конфигурации
	for _, path := range l.configPaths {
		viper.AddConfigPath(path)
	}

	// Настройка переменных окружения
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Устанавливаем значения по умолчанию
	l.setDefaults()

	// Привязываем переменные окружения
	l.bindEnvironmentVariables()

	return nil
}

// readConfig читает файл конфигурации
func (l *ViperConfigLoader) readConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// Файл конфигурации не найден - продолжаем с environment variables и defaults
	}
	return nil
}

// unmarshalConfig преобразует конфигурацию в структуру
func (l *ViperConfigLoader) unmarshalConfig() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// setDefaults устанавливает значения по умолчанию
func (l *ViperConfigLoader) setDefaults() {
	// Настройки сервера
	viper.SetDefault("server.address", defaultServerAddress)
	viper.SetDefault("server.debug", defaultServerDebug)

	// Настройки базы данных
	viper.SetDefault("database.driver", defaultDBDriver)
	viper.SetDefault("database.dsn", defaultDBDSN)

	// Настройки хранилища
	viper.SetDefault("storage.type", defaultStorageType)
	viper.SetDefault("storage.basepath", defaultStorageBasePath)
	viper.SetDefault("storage.s3.region", defaultS3Region)
	viper.SetDefault("storage.s3.bucket", defaultS3Bucket)
	viper.SetDefault("storage.s3.endpoint", "")
	viper.SetDefault("storage.s3.access_key", "")
	viper.SetDefault("storage.s3.secret_key", "")

	// Настройки логирования
	viper.SetDefault("logging.level", defaultLogLevel)
	viper.SetDefault("logging.format", defaultLogFormat)
}

// environmentBinding содержит привязку переменной окружения к ключу конфигурации
type environmentBinding struct {
	configKey string
	envKey    string
}

// bindEnvironmentVariables привязывает переменные окружения к конфигурации
func (l *ViperConfigLoader) bindEnvironmentVariables() {
	bindings := []environmentBinding{
		// Сервер
		{"server.address", "APP_SERVER_ADDRESS"},
		{"server.debug", "APP_SERVER_DEBUG"},

		// База данных
		{"database.driver", "APP_DATABASE_DRIVER"},
		{"database.dsn", "APP_DATABASE_DSN"},

		// Хранилище
		{"storage.type", "APP_STORAGE_TYPE"},
		{"storage.basepath", "APP_STORAGE_BASEPATH"},
		{"storage.s3.region", "APP_STORAGE_S3_REGION"},
		{"storage.s3.bucket", "APP_STORAGE_S3_BUCKET"},
		{"storage.s3.endpoint", "APP_STORAGE_S3_ENDPOINT"},
		{"storage.s3.access_key", "APP_STORAGE_S3_ACCESS_KEY"},
		{"storage.s3.secret_key", "APP_STORAGE_S3_SECRET_KEY"},

		// Логирование
		{"logging.level", "APP_LOGGING_LEVEL"},
		{"logging.format", "APP_LOGGING_FORMAT"},
	}

	for _, binding := range bindings {
		viper.BindEnv(binding.configKey, binding.envKey)
	}
}

// Validator интерфейс для валидации конфигурации
type Validator interface {
	Validate() error
}

// validateConfig проверяет корректность конфигурации
func (l *ViperConfigLoader) validateConfig(cfg Config) error {
	validators := []Validator{
		&serverValidator{cfg.Server},
		&dbValidator{cfg.DB},
		&storageValidator{cfg.Storage},
		&loggingValidator{cfg.Logging},
	}

	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// serverValidator валидатор настроек сервера
type serverValidator struct {
	server Server
}

func (v *serverValidator) Validate() error {
	if v.server.Address == "" {
		return fmt.Errorf("адрес сервера не может быть пустым")
	}
	return nil
}

// dbValidator валидатор настроек базы данных
type dbValidator struct {
	db DB
}

func (v *dbValidator) Validate() error {
	if v.db.Driver == "" {
		return fmt.Errorf("драйвер базы данных не может быть пустым")
	}
	if v.db.DSN == "" {
		return fmt.Errorf("DSN базы данных не может быть пустым")
	}
	return nil
}

// storageValidator валидатор настроек хранилища
type storageValidator struct {
	storage Storage
}

func (v *storageValidator) Validate() error {
	if v.storage.Type != "local" && v.storage.Type != "s3" {
		return fmt.Errorf("тип хранилища должен быть 'local' или 's3', получено: %s", v.storage.Type)
	}

	if v.storage.Type == "local" && v.storage.BasePath == "" {
		return fmt.Errorf("базовый путь не может быть пустым для локального хранилища")
	}

	if v.storage.Type == "s3" {
		if v.storage.S3.Region == "" {
			return fmt.Errorf("регион S3 не может быть пустым")
		}
		if v.storage.S3.Bucket == "" {
			return fmt.Errorf("bucket S3 не может быть пустым")
		}
	}

	return nil
}

// loggingValidator валидатор настроек логирования
type loggingValidator struct {
	logging Logging
}

func (v *loggingValidator) Validate() error {
	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	level := strings.ToLower(v.logging.Level)

	for _, validLevel := range validLevels {
		if level == validLevel {
			return nil
		}
	}

	return fmt.Errorf("неверный уровень логирования: %s. Допустимые уровни: %v", v.logging.Level, validLevels)
}

// IsDevelopment возвращает true, если приложение запущено в режиме разработки
func (c Config) IsDevelopment() bool {
	return c.Server.Debug
}

// IsProduction возвращает true, если приложение запущено в production режиме
func (c Config) IsProduction() bool {
	return !c.Server.Debug
}

// GetDatabaseURL возвращает URL для подключения к базе данных
func (c Config) GetDatabaseURL() string {
	return c.DB.DSN
}

// String возвращает строковое представление конфигурации (без чувствительных данных)
func (c Config) String() string {
	return fmt.Sprintf("Config{Server: %+v, DB: {Driver: %s, DSN: [СКРЫТО]}, Storage: %+v, Logging: %+v}",
		c.Server, c.DB.Driver, c.hideS3Secrets(c.Storage), c.Logging)
}

// hideS3Secrets скрывает чувствительные данные S3 в выводе
func (c Config) hideS3Secrets(storage Storage) Storage {
	if storage.Type == "s3" {
		storage.S3.AccessKey = "[СКРЫТО]"
		storage.S3.SecretKey = "[СКРЫТО]"
	}
	return storage
}
