package notifiers

import (
	"fmt"
	"testing"
	"text/template"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestFormatPolicy(t *testing.T) {
	funcMap := template.FuncMap{
		"header": func(s string) string {
			return fmt.Sprintf("\r\n%v\r\n", s)
		},
		"subheader": func(s string) string {
			return fmt.Sprintf("\r\n\t%v\r\n", s)
		},
		"line": func(s string) string {
			return fmt.Sprintf("%v\r\n", s)
		},
		"list": func(s string) string {
			return fmt.Sprintf("\t - %v\r\n", s)
		},
		"nestedList": func(s string) string {
			return fmt.Sprintf("\t\t - %v\r\n", s)
		},
	}
	alertLink := AlertLink("https://localhost:8080", "alert-id")
	body, err := FormatPolicy(fixtures.GetAlert(), alertLink, funcMap)
	assert.NoError(t, err)
	fmt.Println(body)
}
