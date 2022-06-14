package printer

import (
	"github.com/stackrox/stackrox/pkg/search"
)

const (
	containerNameTemplate = `Container has name '{{.ContainerName}}'`
)

func containerNamePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
	}
	var r resultFields
	var err error
	if r.ContainerName, err = getSingleValueFromFieldMap(search.ContainerName.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(containerNameTemplate, r)
}
