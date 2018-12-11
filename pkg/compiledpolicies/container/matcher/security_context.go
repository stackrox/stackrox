package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	securityContextMatcher "github.com/stackrox/rox/pkg/compiledpolicies/securitycontext/matcher"
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

	return func(container *storage.Container) []*v1.Alert_Violation {
		return matcher(container.GetSecurityContext())
	}, nil
}
