package repository

// QueryExecutor выполняет SQL-запросы и возвращает строки результата.
type QueryExecutor interface {
	Execute(query string, args ...any) ([]map[string]any, error)
}
