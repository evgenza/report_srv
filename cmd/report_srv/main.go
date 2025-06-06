package main

import (
	"context"
	"log"

	"report_srv/internal/config"
	sqlinfra "report_srv/internal/infrastructure/sql"
	"report_srv/internal/infrastructure/storage"
	"report_srv/internal/infrastructure/template"
	"report_srv/internal/usecase"
)

func main() {
	// Example configuration. In real application this would be loaded from
	// file or environment.
	cfg := config.Config{
		Driver: "postgres",
		DSN:    "postgres://user:pass@localhost/db?sslmode=disable",
	}

	db, err := sqlinfra.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	svc := usecase.ReportService{
		Executor: db,
		Filler:   template.XLSXFiller{}, // or DOCXFiller depending on template
		Storage:  storage.S3Storage{BasePath: "./templates"},
		Reports:  sqlinfra.ReportRepository{DB: db.DB},
	}

	if _, err := svc.Generate(context.Background(), "sample-report"); err != nil {
		log.Fatal(err)
	}
}
