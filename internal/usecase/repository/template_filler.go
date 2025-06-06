package repository

import (
	"context"

	"report_srv/internal/domain/report"
)

// TemplateFiller заполняет шаблон переданными данными.
type TemplateFiller interface {
	Fill(tmpl []byte, data []map[string]any) ([]byte, error)
}

// TemplateStorage предоставляет доступ к файлам шаблонов (например, из S3).
type TemplateStorage interface {
	Download(key string) ([]byte, error)
}

// ReportRepository загружает метаданные отчётов, описывающие расположение шаблонов и SQL-запросов.
type ReportRepository interface {
	GetByID(ctx context.Context, id string) (report.Report, error)
}
