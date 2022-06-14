package printer

import (
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	resourceTemplate = `{{.Name}} set to {{.Value}} {{.Unit}}{{if .ContainerName}} for container '{{.ContainerName}}'{{end}}`
)

func resourcePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Name          string
		Value         string
		Unit          string
	}
	r := make([]resultFields, 0)
	if cpuCoresLimit, err := getSingleValueFromFieldMap(search.CPUCoresLimit.String(), fieldMap); err == nil {
		r = append(r, resultFields{Name: "CPU limit", Value: cpuCoresLimit, Unit: "cores"})
	}
	if cpuCoresRequest, err := getSingleValueFromFieldMap(search.CPUCoresRequest.String(), fieldMap); err == nil {
		r = append(r, resultFields{Name: "CPU request", Value: cpuCoresRequest, Unit: "cores"})
	}
	if memRequest, err := getSingleValueFromFieldMap(search.MemoryRequest.String(), fieldMap); err == nil {
		r = append(r, resultFields{Name: "Memory request", Value: memRequest, Unit: "MB"})
	}
	if memLimit, err := getSingleValueFromFieldMap(search.MemoryLimit.String(), fieldMap); err == nil {
		r = append(r, resultFields{Name: "Memory limit", Value: memLimit, Unit: "MB"})
	}
	if containerName, err := getSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap); err == nil {
		for i := range r {
			r[i].ContainerName = containerName
		}
	}
	messages := make([]string, 0, len(r))
	for _, values := range r {
		msg, err := executeTemplate(resourceTemplate, values)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg...)
	}
	return messages, nil
}
