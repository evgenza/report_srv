package sql

import (
	"context"
	"database/sql"

	"report_srv/internal/domain/report"
)

// ReportRepository retrieves report metadata from the database.
type ReportRepository struct {
	DB *sql.DB
}

// GetByID loads a report by its ID.
func (r ReportRepository) GetByID(ctx context.Context, id string) (report.Report, error) {
	var rep report.Report
	err := r.DB.QueryRowContext(ctx, `SELECT id, template_type, template_key FROM reports WHERE id = $1`, id).
		Scan(&rep.ID, &rep.Template, &rep.TemplateKey)
	if err != nil {
		return report.Report{}, err
	}

	rows, err := r.DB.QueryContext(ctx, `SELECT query_sql FROM report_queries WHERE report_id = $1 ORDER BY seq`, id)
	if err != nil {
		return report.Report{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return report.Report{}, err
		}
		rep.Queries = append(rep.Queries, q)
	}
	return rep, nil
}
