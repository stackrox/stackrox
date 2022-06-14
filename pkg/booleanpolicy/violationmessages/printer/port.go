package printer

import (
	"github.com/stackrox/stackrox/pkg/search"
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

const (
	nodePortTemplate = `Exposed node port {{.ExposedNodePort}}{{if .Protocol}}/{{.Protocol}}{{end}} is present`
)

func nodePortPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ExposedNodePort string
		Protocol        string
	}
	r := resultFields{}
	r.ExposedNodePort = maybeGetSingleValueFromFieldMap(search.ExposedNodePort.String(), fieldMap)
	r.Protocol = maybeGetSingleValueFromFieldMap(search.PortProtocol.String(), fieldMap)
	return executeTemplate(nodePortTemplate, r)
}
