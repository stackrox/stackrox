package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	containerMatcher "github.com/stackrox/rox/pkg/compiledpolicies/container/matcher"
)

func init() {
	compilers = append(compilers, newContainerMatcher)
}

func newContainerMatcher(policy *storage.Policy) (Matcher, error) {
	matcher, err := containerMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(deployment *storage.Deployment) []*v1.Alert_Violation {
		var violations []*v1.Alert_Violation
		for _, container := range deployment.GetContainers() {
			violations = append(violations, matcher(container)...)
		}
		return violations
	}, nil
}
