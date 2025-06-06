package query

// Query описывает SQL-запрос, который нужно выполнить для заполнения отчёта.
type Query struct {
	ID     string
	SQL    string
	Params []any
}
