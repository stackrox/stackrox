package templates

import (
	"path/filepath"
	"text/template"
)

const templatePath = "/data/templates"

// ReadFileAndTemplate reads and renders the template for the file
func ReadFileAndTemplate(path string) (*template.Template, error) {
	return template.ParseFiles(filepath.Join(templatePath, path))
}
