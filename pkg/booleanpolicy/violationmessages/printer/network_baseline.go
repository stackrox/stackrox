package printer

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

const (
	networkBaselineTemplate = `{{if .NotInNetworkBaseline}}Unexpected{{else}}Expected{{end}} network flow found in deployment.{{if .SrcName}} Source name: '{{.SrcName}}'.{{end}}{{if .DstName}} Destination name: '{{.DstName}}'.{{end}}{{if .DstPort}} Destination port: '{{.DstPort}}'.{{end}}{{if .Protocol}} Protocol: '{{.Protocol}}'.{{end}}`
)

func networkBaselinePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		SrcName              string
		DstName              string
		DstPort              string
		Protocol             string
		NotInNetworkBaseline bool
	}
	r := resultFields{}
	r.SrcName = maybeGetSingleValueFromFieldMap(augmentedobjs.NetworkFlowSrcNameCustomTag, fieldMap)
	r.DstName = maybeGetSingleValueFromFieldMap(augmentedobjs.NetworkFlowDstNameCustomTag, fieldMap)
	r.DstPort = maybeGetSingleValueFromFieldMap(augmentedobjs.NetworkFlowDstPortCustomTag, fieldMap)
	// Protocol value matched gets converted to lowercase. Capitalize it
	r.Protocol = strings.ToUpper(maybeGetSingleValueFromFieldMap(augmentedobjs.NetworkFlowL4Protocol, fieldMap))
	notInNetworkBaseline, err := getSingleValueFromFieldMap(augmentedobjs.NotInNetworkBaselineCustomTag, fieldMap)
	if err != nil {
		return nil, err
	}
	if r.NotInNetworkBaseline, err = strconv.ParseBool(notInNetworkBaseline); err != nil {
		return nil, err
	}
	return executeTemplate(networkBaselineTemplate, r)
}
