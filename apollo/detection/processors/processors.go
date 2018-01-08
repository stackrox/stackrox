package processors

import (
	"regexp"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

var (
	// PolicyCategoryCompiler is a map of categories to their corresponding compiler function.
	PolicyCategoryCompiler = map[v1.Policy_Category]func(*v1.Policy) (CompiledPolicy, error){}
)

// CompiledPolicy allows matching against a container in a deployment.
type CompiledPolicy interface {
	Match(*v1.Deployment, *v1.Container) []*v1.Alert_Violation
}

// CompileStringRegex returns the compiled regex if string is not empty,
// otherwise nil is returned.
func CompileStringRegex(policy string) (*regexp.Regexp, error) {
	if policy == "" {
		return nil, nil
	}
	return regexp.Compile(policy)
}
