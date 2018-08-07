package matcher

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	configMatcher "bitbucket.org/stack-rox/apollo/pkg/compiledpolicies/containerconfig/matcher"
)

func init() {
	compilers = append(compilers, newConfigMatcher)
}

func newConfigMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := configMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *v1.Container) []*v1.Alert_Violation {
		return matcher(container.GetConfig())
	}, nil
}
