package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	hostPidTemplate = `Deployment {{- if .HostPID }} uses the host's process ID namespace{{else}} does not use the host's process ID namespace{{end}}`
)

func hostPIDPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		HostPID       bool
	}

	r := resultFields{}
	var err error
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	hostPID, err := getSingleValueFromFieldMap(search.HostPID.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.HostPID, err = strconv.ParseBool(hostPID); err != nil {
		return nil, err
	}
	return executeTemplate(hostPidTemplate, r)
}
