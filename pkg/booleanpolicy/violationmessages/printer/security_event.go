package printer

import (
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

const (
	securityEventTemplate = `Security event reported by '{{.Source}}'`
)

func securityEventPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		Source string
	}
	r := resultFields{}
	var err error
	if r.Source, err = getSingleValueFromFieldMap(augmentedobjs.SecurityEventSourceCustomTag, fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(securityEventTemplate, r)
}
