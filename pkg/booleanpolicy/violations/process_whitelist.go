package violations

import (
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

func processWhitelistPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	msgTemplate := `{{if .NotWhitelisted}}Unexpected{{else}}Expected{{end}} process{{if .ProcessName}} '{{.ProcessName}}'{{end}} in container{{if .ContainerName}} '{{.ContainerName}}'{{end}}`
	type resultFields struct {
		ContainerName  string
		ProcessName    string
		NotWhitelisted bool
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.ProcessName = maybeGetSingleValueFromFieldMap(search.ProcessName.String(), fieldMap)
	notWhitelisted, err := getSingleValueFromFieldMap(augmentedobjs.NotWhitelistedCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	if r.NotWhitelisted, err = strconv.ParseBool(notWhitelisted); err != nil {
		return nil, err
	}
	return executeTemplate(msgTemplate, r)
}
