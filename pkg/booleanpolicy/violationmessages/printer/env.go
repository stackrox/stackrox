package printer

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
)

var (
	envVarSourceToNameMap = map[string]string{
		"RAW":            "",
		"SECRET_KEY":     "Secret",
		"CONFIG_MAP_KEY": "ConfigMap",
		"FIELD":          "field",
		"RESOURCE_FIELD": "resource field"}
)

const (
	envTemplate = `Environment variable '{{.Name}}' is present
	{{- if .ContainerName}} in container '{{.ContainerName}}'{{end}}
	{{- if .Source}} and references a {{.Source}}{{end}}`
)

func envPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Source        string
		Name          string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	fieldValue, err := getSingleValueFromFieldMap(augmentedobjs.EnvironmentVarCustomTag, fieldMap)
	if err != nil {
		return nil, errors.New("invalid env var in result")
	}
	envVar := strings.SplitN(fieldValue, augmentedobjs.CompositeFieldCharSep, 3)
	if len(envVar) != 3 {
		return nil, errors.New("failed to parse env var result")
	}
	r.Source = envVarSourceToNameMap[strings.ToUpper(envVar[0])]
	r.Name = envVar[1]
	return executeTemplate(envTemplate, r)
}
