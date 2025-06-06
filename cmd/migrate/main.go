package main

import (
	"log"
	"os"

	"report_srv/internal/database"
)

func main() {
	// Create database connection
	cfg := &database.Config{
		Driver: os.Getenv("APP_DATABASE_DRIVER"),
		DSN:    os.Getenv("APP_DATABASE_DSN"),
		Debug:  true,
	}

	db, err := database.NewDatabase(*cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")
}
