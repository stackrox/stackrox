package templates

import (
	"bytes"
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

// ExecuteToBytes executes the given template and returns the result as a []byte.
func ExecuteToBytes(tmpl *template.Template, data interface{}) ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
