package printer

import (
	"strconv"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	hostNetworkTemplate = `Deployment {{- if .HostNetwork }} uses the host's network namespace{{else}} does not use the host's network namespace{{end}}`
)

func hostNetworkPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		HostNetwork   bool
	}

	r := resultFields{}
	var err error
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	hostNetwork, err := getSingleValueFromFieldMap(search.HostNetwork.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.HostNetwork, err = strconv.ParseBool(hostNetwork); err != nil {
		return nil, err
	}
	return executeTemplate(hostNetworkTemplate, r)
}
