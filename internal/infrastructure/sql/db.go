package sql

import (
    "database/sql"
)

// DB wraps *sql.DB to satisfy the QueryExecutor interface.
type DB struct {
    *sql.DB
}

// Execute executes a query and returns rows as a slice of map.
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
