package bootstrapcommon

import (
	"bytes"
	"os"
	"text/template"
)

// NewTemplate returns a factory that creates a named template from the given source.
func NewTemplate(tpl string) func(name string) *template.Template {
	return func(name string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Parse(tpl))
	}
}

// RenderFile executes a template with the given data and writes the result to filePath.
func RenderFile(templateMap map[string]interface{}, temp func(s string) *template.Template, filePath string) error {
	buf := bytes.NewBuffer(nil)
	if err := temp(filePath).Execute(buf, templateMap); err != nil {
		return err
	}
	return os.WriteFile(filePath, buf.Bytes(), 0644)
}
