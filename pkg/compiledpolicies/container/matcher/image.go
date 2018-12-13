package matcher

import (
	"github.com/stackrox/rox/generated/storage"
	imageMatcher "github.com/stackrox/rox/pkg/compiledpolicies/image/matcher"
)

func init() {
	compilers = append(compilers, newImageMatcher)
}

func newImageMatcher(policy *storage.Policy) (Matcher, error) {
	matcher, err := imageMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *storage.Container) []*storage.Alert_Violation {
		return matcher(container.GetImage())
	}, nil
}
