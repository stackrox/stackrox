package deploy

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

type fileFormatWrapper struct {
	DeploymentFormat *v1.DeploymentFormat
}

var deploymentFormatMap = map[string]v1.DeploymentFormat{
	"kubectl": v1.DeploymentFormat_KUBECTL,
	"helm":    v1.DeploymentFormat_HELM,
}

func (f *fileFormatWrapper) String() string {
	return strings.ToLower(f.DeploymentFormat.String())
}

func (f *fileFormatWrapper) Set(input string) error {
	val, _ := v1.DeploymentFormat_value[strings.ToUpper(input)]
	*f.DeploymentFormat = v1.DeploymentFormat(val)
	return nil
}

func (f *fileFormatWrapper) Type() string {
	return "output format"
}
