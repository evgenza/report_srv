package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Server содержит настройки HTTP-сервера.
type Server struct {
	Address string `mapstructure:"address"`
	Debug   bool   `mapstructure:"debug"`
}

// DB содержит параметры подключения к БД.
type DB struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// Storage описывает настройки хранилища файлов.
type Storage struct {
	Type     string `mapstructure:"type"`
	BasePath string `mapstructure:"basepath"`
	S3       S3     `mapstructure:"s3"`
}

// S3 содержит настройки для S3-совместимого хранилища.
type S3 struct {
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// Logging содержит настройки логирования.
type Logging struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Config объединяет все разделы конфигурации.
type Config struct {
	Server  Server  `mapstructure:"server"`
	DB      DB      `mapstructure:"database"`
	Storage Storage `mapstructure:"storage"`
	Logging Logging `mapstructure:"logging"`
}

// Load читает конфигурацию из файла и окружения с помощью viper.
func Load() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/report-service")

	// Настройка для environment variables
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Значения по умолчанию
	setDefaults()

	// Привязка environment variables к конфигурации
	bindEnvironmentVariables()

	// Чтение файла конфигурации (опционально)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("failed to read config file: %w", err)
		}
		// Если файл конфигурации не найден, продолжаем с environment variables и defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		return Config{}, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.debug", true)

	// Database defaults
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.dsn", "postgres://user:pass@localhost:5432/reports?sslmode=disable")

	// Storage defaults
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.basepath", "./templates")
	viper.SetDefault("storage.s3.region", "us-east-1")
	viper.SetDefault("storage.s3.bucket", "report-srv-bucket")
	viper.SetDefault("storage.s3.endpoint", "")
	viper.SetDefault("storage.s3.access_key", "")
	viper.SetDefault("storage.s3.secret_key", "")

	// Logging defaults
	viper.SetDefault("logging.level", "debug")
	viper.SetDefault("logging.format", "text")
}

// bindEnvironmentVariables привязывает переменные окружения к конфигурации
func bindEnvironmentVariables() {
	// Server
	viper.BindEnv("server.address", "APP_SERVER_ADDRESS")
	viper.BindEnv("server.debug", "APP_SERVER_DEBUG")

	// Database
	viper.BindEnv("database.driver", "APP_DATABASE_DRIVER")
	viper.BindEnv("database.dsn", "APP_DATABASE_DSN")

	// Storage
	viper.BindEnv("storage.type", "APP_STORAGE_TYPE")
	viper.BindEnv("storage.basepath", "APP_STORAGE_BASEPATH")
	viper.BindEnv("storage.s3.region", "APP_STORAGE_S3_REGION")
	viper.BindEnv("storage.s3.bucket", "APP_STORAGE_S3_BUCKET")
	viper.BindEnv("storage.s3.endpoint", "APP_STORAGE_S3_ENDPOINT")
	viper.BindEnv("storage.s3.access_key", "APP_STORAGE_S3_ACCESS_KEY")
	viper.BindEnv("storage.s3.secret_key", "APP_STORAGE_S3_SECRET_KEY")

	// Logging
	viper.BindEnv("logging.level", "APP_LOGGING_LEVEL")
	viper.BindEnv("logging.format", "APP_LOGGING_FORMAT")
}

// validateConfig проверяет корректность конфигурации
func validateConfig(cfg Config) error {
	// Проверка адреса сервера
	if cfg.Server.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	// Проверка настроек базы данных
	if cfg.DB.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}

	if cfg.DB.DSN == "" {
		return fmt.Errorf("database DSN cannot be empty")
	}

	// Проверка настроек хранилища
	if cfg.Storage.Type != "local" && cfg.Storage.Type != "s3" {
		return fmt.Errorf("storage type must be 'local' or 's3', got: %s", cfg.Storage.Type)
	}

	if cfg.Storage.Type == "local" && cfg.Storage.BasePath == "" {
		return fmt.Errorf("storage basepath cannot be empty for local storage")
	}

	if cfg.Storage.Type == "s3" {
		if cfg.Storage.S3.Region == "" {
			return fmt.Errorf("S3 region cannot be empty")
		}
		if cfg.Storage.S3.Bucket == "" {
			return fmt.Errorf("S3 bucket cannot be empty")
		}
	}

	// Проверка уровня логирования
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	isValidLevel := false
	for _, level := range validLogLevels {
		if strings.ToLower(cfg.Logging.Level) == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("invalid logging level: %s. Valid levels: %v", cfg.Logging.Level, validLogLevels)
	}

	return nil
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
	return fmt.Sprintf("Config{Server: %+v, DB: {Driver: %s, DSN: [HIDDEN]}, Storage: %+v, Logging: %+v}",
		c.Server, c.DB.Driver, c.Storage, c.Logging)
}
