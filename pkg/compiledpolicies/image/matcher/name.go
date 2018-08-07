package matcher

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	imageNameMatcher "bitbucket.org/stack-rox/apollo/pkg/compiledpolicies/imagename/matcher"
)

func init() {
	compilers = append(compilers, newNameMatcher)
}

func newNameMatcher(policy *v1.Policy) (Matcher, error) {
	nameMatcher, err := imageNameMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if nameMatcher == nil {
		return nil, nil
	}

	return func(image *v1.Image) []*v1.Alert_Violation {
		return nameMatcher(image.GetName())
	}, nil
}
