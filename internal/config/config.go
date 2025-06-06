package config

import (
	"bufio"
	"os"
	"strings"
)

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

// loadFromFile читает файл .env и устанавливает переменные окружения,
// если они ещё не заданы.
func loadFromFile() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if _, ok := os.LookupEnv(key); !ok {
			os.Setenv(key, val)
		}
	}
}

// Load считывает настройки из переменных окружения.
// Перед чтением переменных предпринимается попытка загрузить их из файла .env,
// чтобы локальная конфигурация могла использовать единый источник данных.
//
//	DB_DRIVER   - название SQL‑драйвера
//	DB_DSN      - строка подключения к базе данных
//	S3_BASEPATH - путь к хранилищу шаблонов
//
// Переменные окружения необязательны: если они не заданы, используются нулевые значения.
func Load() Config {
	loadFromFile()
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
