package deploy

import (
	"fmt"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

type fileFormatWrapper struct {
	DeploymentFormat *v1.DeploymentFormat
}

func (f *fileFormatWrapper) String() string {
	return strings.ToLower(f.DeploymentFormat.String())
}

func (f *fileFormatWrapper) Set(input string) error {
	val, ok := v1.DeploymentFormat_value[strings.ToUpper(input)]
	if !ok {
		return fmt.Errorf("%q is not a valid option", input)
	}
	*f.DeploymentFormat = v1.DeploymentFormat(val)
	return nil
}

func (f *fileFormatWrapper) Type() string {
	return "output format"
}
