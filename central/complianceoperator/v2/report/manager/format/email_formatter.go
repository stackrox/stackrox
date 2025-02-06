package format

import (
	"text/template"

	"github.com/stackrox/rox/pkg/templates"
)

//go:generate mockgen-wrapper
type EmailFormatter interface {
	FormatWithDetails(string, string, any) (string, error)
}

type emailFormatterImpl struct{}

func NewEmailFormatter() EmailFormatter {
	return &emailFormatterImpl{}
}

func (f *emailFormatterImpl) FormatWithDetails(templateName string, format string, data any) (string, error) {
	tmpl, err := template.New(templateName).Parse(format)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}
