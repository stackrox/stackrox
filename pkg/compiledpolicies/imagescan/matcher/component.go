package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	componentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/imagescancomponent/matcher"
)

func init() {
	compilers = append(compilers, newComponentMatcher)
}

func newComponentMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := componentMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(scan *v1.ImageScan) []*v1.Alert_Violation {
		var violations []*v1.Alert_Violation
		for _, component := range scan.GetComponents() {
			violations = append(violations, matcher(component)...)
		}
		return violations
	}, nil
}
