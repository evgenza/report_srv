package repository

// QueryExecutor executes SQL queries and returns resulting rows.
type QueryExecutor interface {
    Execute(query string, args ...any) ([]map[string]any, error)
}
