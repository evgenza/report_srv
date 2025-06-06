package template

import (
    "io/ioutil"
)

// DOCXFiller implements TemplateFiller for docx templates.
type DOCXFiller struct{}

// Fill returns contents of the template without modification.
func (DOCXFiller) Fill(templatePath string, data []map[string]any) ([]byte, error) {
    return ioutil.ReadFile(templatePath)
}
