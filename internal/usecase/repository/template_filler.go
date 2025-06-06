package repository

import (
	"context"

	"report_srv/internal/domain/report"
)

// TemplateFiller fills template bytes using provided data.
type TemplateFiller interface {
	Fill(tmpl []byte, data []map[string]any) ([]byte, error)
}

// TemplateStorage provides access to template files (e.g. from S3).
type TemplateStorage interface {
	Download(key string) ([]byte, error)
}

// ReportRepository loads report metadata describing where to find templates and queries.
type ReportRepository interface {
	GetByID(ctx context.Context, id string) (report.Report, error)
}
