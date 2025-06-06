package main

import (
	"context"
	"log"

	"report_srv/internal/infrastructure/sql"
	"report_srv/internal/infrastructure/template"
	"report_srv/internal/usecase"
)

func main() {
	// This is a stub main demonstrating wiring of the service.
	svc := usecase.ReportService{
		Executor: &sql.DB{},             // TODO: initialize with actual *sql.DB
		Filler:   template.XLSXFiller{}, // or DOCXFiller depending on template
	}

	if _, err := svc.Generate(context.Background(), "template.xlsx", []string{"SELECT 1"}); err != nil {
		log.Fatal(err)
	}
}
