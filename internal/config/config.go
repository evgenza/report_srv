package config

// Config represents application configuration.
type Config struct {
	Driver string // name of SQL driver, e.g. "postgres", "mysql"
	DSN    string // database connection string
}
