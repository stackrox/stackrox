package violations

import (
	"github.com/stackrox/rox/pkg/search"
)

func portPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	msgTemplate := `Exposed port {{.Port}}{{if .Protocol}}/{{.Protocol}}{{end}} is present`
	type resultFields struct {
		Port     string
		Protocol string
	}
	r := resultFields{}
	r.Port = maybeGetSingleValueFromFieldMap(search.Port.String(), fieldMap)
	r.Protocol = maybeGetSingleValueFromFieldMap(search.PortProtocol.String(), fieldMap)
	return executeTemplate(msgTemplate, r)
}
