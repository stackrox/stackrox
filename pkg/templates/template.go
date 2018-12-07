package templates

import (
	"path/filepath"
	"strings"
	"text/template"
)

const templatePath = "/data/templates"

// ReadFileAndTemplate reads and renders the template for the file
func ReadFileAndTemplate(path string) (*template.Template, error) {
	return template.ParseFiles(filepath.Join(templatePath, path))
}

// ExecuteToString executes the given template and returns the result as a string.
func ExecuteToString(tmpl *template.Template, data interface{}) (string, error) {
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
