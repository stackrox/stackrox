package violationmessages

import (
	"github.com/stackrox/rox/pkg/search"
)

const (
	portTemplate = `Exposed port {{.Port}}{{if .Protocol}}/{{.Protocol}}{{end}} is present`
)

func portPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		Port     string
		Protocol string
	}
	r := resultFields{}
	r.Port = maybeGetSingleValueFromFieldMap(search.Port.String(), fieldMap)
	r.Protocol = maybeGetSingleValueFromFieldMap(search.PortProtocol.String(), fieldMap)
	return executeTemplate(portTemplate, r)
}
