package query

// Query encapsulates SQL statement that should be executed to fill the report.
type Query struct {
    ID    string
    SQL   string
    Params []any
}
