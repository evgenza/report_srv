package template

// DOCXFiller implements TemplateFiller for docx templates.
type DOCXFiller struct{}

// Fill returns contents of the template without modification.
func (DOCXFiller) Fill(tmpl []byte, data []map[string]any) ([]byte, error) {
	// TODO: implement real docx template filling
	return tmpl, nil
}
