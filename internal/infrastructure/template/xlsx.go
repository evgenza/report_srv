package template

// XLSXFiller implements TemplateFiller for xlsx templates.
type XLSXFiller struct{}

// Fill returns contents of the template without modification.
func (XLSXFiller) Fill(tmpl []byte, data []map[string]any) ([]byte, error) {
	// TODO: implement real xlsx template filling
	return tmpl, nil
}
