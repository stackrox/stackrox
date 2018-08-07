package matcher

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	imageMatcher "bitbucket.org/stack-rox/apollo/pkg/compiledpolicies/image/matcher"
)

func init() {
	compilers = append(compilers, newImageMatcher)
}

func newImageMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := imageMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *v1.Container) []*v1.Alert_Violation {
		return matcher(container.GetImage())
	}, nil
}
