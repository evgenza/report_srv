package template

// DOCXFiller реализует TemplateFiller для шаблонов docx.
type DOCXFiller struct{}

// Fill возвращает содержимое шаблона без изменений.
func (DOCXFiller) Fill(tmpl []byte, data []map[string]any) ([]byte, error) {
	// TODO: реализовать заполнение шаблона docx
	return tmpl, nil
}
