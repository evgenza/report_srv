package repository

// TemplateFiller fills template files using provided data.
type TemplateFiller interface {
    Fill(templatePath string, data []map[string]any) ([]byte, error)
}
