package config

import (
	"github.com/spf13/viper"
)

// Server содержит настройки HTTP-сервера.
type Server struct {
	Address string `mapstructure:"address"`
}

// DB содержит параметры подключения к БД.
type DB struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// Storage описывает путь к шаблонам.
type Storage struct {
	BasePath string `mapstructure:"basepath"`
}

// Config объединяет все разделы конфигурации.
type Config struct {
	Server  Server  `mapstructure:"server"`
	DB      DB      `mapstructure:"database"`
	Storage Storage `mapstructure:"storage"`
}

// Load читает конфигурацию из файла и окружения с помощью viper.
func Load() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// Значения по умолчанию
	viper.SetDefault("server.address", ":8080")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
