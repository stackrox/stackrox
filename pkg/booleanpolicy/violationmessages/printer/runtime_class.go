package printer

import (
	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
)

const (
	runtimeClassTemplate = `Runtime Class is set to '{{.RuntimeClass}}'`
)

func runtimeClassPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		RuntimeClass string
	}

	runtimeClass, err := getSingleValueFromFieldMap(augmentedobjs.RuntimeClassCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	return executeTemplate(runtimeClassTemplate, &resultFields{RuntimeClass: runtimeClass})
}
