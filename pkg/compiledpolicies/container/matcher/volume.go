package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	volumeMatcher "github.com/stackrox/rox/pkg/compiledpolicies/volume/matcher"
)

func init() {
	compilers = append(compilers, newVolumeMatcher)
}

func newVolumeMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := volumeMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *v1.Container) []*v1.Alert_Violation {
		var violations []*v1.Alert_Violation
		for _, volume := range container.GetVolumes() {
			violations = append(violations, matcher(volume)...)
		}
		return violations
	}, nil
}
