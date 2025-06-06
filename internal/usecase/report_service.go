package usecase

import (
	"context"

	"report_srv/internal/usecase/repository"
)

// ReportService generates reports using provided template and SQL queries.
type ReportService struct {
	Executor repository.QueryExecutor
	Filler   repository.TemplateFiller
	Storage  repository.TemplateStorage
	Reports  repository.ReportRepository
}

// Generate executes queries and fills template.
func (s *ReportService) Generate(ctx context.Context, reportID string) ([]byte, error) {
	rep, err := s.Reports.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	tmpl, err := s.Storage.Download(rep.TemplateKey)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for _, q := range rep.Queries {
		rows, err := s.Executor.Execute(q)
		if err != nil {
			return nil, err
		}
		results = append(results, rows...)
	}

	return s.Filler.Fill(tmpl, results)
}
