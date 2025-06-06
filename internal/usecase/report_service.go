package usecase

import (
	"context"

	"report_srv/internal/usecase/repository"
)

// ReportService generates reports using provided template and SQL queries.
type ReportService struct {
	Executor repository.QueryExecutor
	Filler   repository.TemplateFiller
}

// Generate executes queries and fills template.
func (s *ReportService) Generate(ctx context.Context, templatePath string, queries []string) ([]byte, error) {
	var results []map[string]any
	for _, q := range queries {
		rows, err := s.Executor.Execute(q)
		if err != nil {
			return nil, err
		}
		results = append(results, rows...)
	}
	return s.Filler.Fill(templatePath, results)
}
