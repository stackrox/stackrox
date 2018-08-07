package matcher

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	securityContextMatcher "bitbucket.org/stack-rox/apollo/pkg/compiledpolicies/securitycontext/matcher"
)

func init() {
	compilers = append(compilers, newSecurityContextMatcher)
}

func newSecurityContextMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := securityContextMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *v1.Container) []*v1.Alert_Violation {
		return matcher(container.GetSecurityContext())
	}, nil
}
