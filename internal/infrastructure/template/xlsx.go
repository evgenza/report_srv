package template

// XLSXFiller реализует TemplateFiller для шаблонов xlsx.
type XLSXFiller struct{}

// Fill возвращает содержимое шаблона без изменений.
func (XLSXFiller) Fill(tmpl []byte, data []map[string]any) ([]byte, error) {
	// TODO: реализовать заполнение шаблона xlsx
	return tmpl, nil
}
