package template

import (
    "io/ioutil"
)

// XLSXFiller implements TemplateFiller for xlsx templates.
type XLSXFiller struct{}

// Fill returns contents of the template without modification.
func (XLSXFiller) Fill(templatePath string, data []map[string]any) ([]byte, error) {
    return ioutil.ReadFile(templatePath)
}
