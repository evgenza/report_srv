package sql

import (
	"database/sql"
)

// DB оборачивает *sql.DB и реализует интерфейс QueryExecutor.
type DB struct {
	*sql.DB
}

// Open создаёт новое подключение к базе данных с указанным драйвером и DSN.
func Open(driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{DB: db}, nil
}

// Execute выполняет запрос и возвращает строки в виде среза map.
func (d *DB) Execute(query string, args ...any) ([]map[string]any, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range ptrs {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		rowMap := make(map[string]any)
		for i, col := range cols {
			rowMap[col] = vals[i]
		}
		results = append(results, rowMap)
	}
	return results, nil
}
