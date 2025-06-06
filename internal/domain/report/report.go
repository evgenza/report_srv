package report

// TemplateType обозначает поддерживаемые типы шаблонов.
type TemplateType int

const (
	TemplateXLSX TemplateType = iota
	TemplateDOCX
)

// Report содержит данные о шаблоне отчёта.
type Report struct {
	ID          string
	Template    TemplateType
	TemplateKey string
	Queries     []string
}
