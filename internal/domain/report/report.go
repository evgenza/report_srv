package report

// TemplateType represents the type of supported templates.
type TemplateType int

const (
    TemplateXLSX TemplateType = iota
    TemplateDOCX
)

// Report holds information about a report template.
type Report struct {
    ID          string
    Template    TemplateType
    TemplatePath string
    Queries     []string
}
