package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	hostIPCTemplate = `Deployment {{- if .HostIPC }} uses the host's IPC namespace{{else}} does not use the host's IPC namespace{{end}}`
)

func hostIPCPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		HostIPC       bool
	}

	r := resultFields{}
	var err error
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	hostIPC, err := getSingleValueFromFieldMap(search.HostIPC.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.HostIPC, err = strconv.ParseBool(hostIPC); err != nil {
		return nil, err
	}
	return executeTemplate(hostIPCTemplate, r)
}
