package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	baselineTemplate = `{{if .NotInBaseline}}Unexpected{{else}}Expected{{end}} process{{if .ProcessName}} '{{.ProcessName}}'{{end}} in container{{if .ContainerName}} '{{.ContainerName}}'{{end}}`
)

func processBaselinePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ProcessName   string
		NotInBaseline bool
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.ProcessName = maybeGetSingleValueFromFieldMap(search.ProcessName.String(), fieldMap)
	notInBaseline, err := getSingleValueFromFieldMap(augmentedobjs.NotInProcessBaselineCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	if r.NotInBaseline, err = strconv.ParseBool(notInBaseline); err != nil {
		return nil, err
	}
	return executeTemplate(baselineTemplate, r)
}
