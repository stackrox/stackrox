package matcher

import (
	"github.com/stackrox/rox/generated/storage"
	componentMatcher "github.com/stackrox/rox/pkg/compiledpolicies/imagescancomponent/matcher"
)

func init() {
	compilers = append(compilers, newComponentMatcher)
}

func newComponentMatcher(policy *storage.Policy) (Matcher, error) {
	matcher, err := componentMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(scan *storage.ImageScan) []*storage.Alert_Violation {
		var violations []*storage.Alert_Violation
		for _, component := range scan.GetComponents() {
			violations = append(violations, matcher(component)...)
		}
		return violations
	}, nil
}
