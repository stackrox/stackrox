package processors

import (
	"regexp"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

var (
	// PolicySegmentCompilers is a list of policy compiler function.
	PolicySegmentCompilers []func(*v1.Policy) (CompiledPolicy, error)
)

// CompiledPolicy allows matching against a container in a deployment.
type CompiledPolicy interface {
	Match(*v1.Deployment, *v1.Container) ([]*v1.Alert_Violation, bool)
}

// CompileStringRegex returns the compiled regex if string is not empty,
// otherwise nil is returned.
func CompileStringRegex(policy string) (*regexp.Regexp, error) {
	if policy == "" {
		return nil, nil
	}
	return regexp.Compile(policy)
}
