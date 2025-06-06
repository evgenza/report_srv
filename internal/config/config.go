package config

import "os"

// DB holds settings required to connect to the database.
type DB struct {
	Driver string // e.g. "postgres", "mysql", "sqlite3"
	DSN    string // complete database connection string
}

// Storage describes where templates are stored.
type Storage struct {
	BasePath string // path to the local directory or bucket used for templates
}

// Config contains all configuration required by the service.
type Config struct {
	DB      DB
	Storage Storage
}

// Load reads configuration from environment variables.
//
//	DB_DRIVER   - SQL driver name
//	DB_DSN      - database connection string
//	S3_BASEPATH - path to the template storage location
//
// Environment variables are optional; zero values will be used if they are not set.
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
