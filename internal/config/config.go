package config

import "os"

// DB содержит настройки для подключения к базе данных.
type DB struct {
	Driver string // например "postgres", "mysql", "sqlite3"
	DSN    string // полная строка подключения к БД
}

// Storage описывает местоположение шаблонов.
type Storage struct {
	BasePath string // путь к каталогу или бакету с шаблонами
}

// Config содержит все необходимые сервису настройки.
type Config struct {
	DB      DB
	Storage Storage
}

// Load считывает настройки из переменных окружения.
//
//	DB_DRIVER   - название SQL‑драйвера
//	DB_DSN      - строка подключения к базе данных
//	S3_BASEPATH - путь к хранилищу шаблонов
//
// Переменные окружения необязательны: если они не заданы, используются нулевые значения.
func Load() Config {
	return Config{
		DB: DB{
			Driver: os.Getenv("DB_DRIVER"),
			DSN:    os.Getenv("DB_DSN"),
		},
		Storage: Storage{
			BasePath: os.Getenv("S3_BASEPATH"),
		},
	}
}
