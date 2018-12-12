package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	imageNameMatcher "github.com/stackrox/rox/pkg/compiledpolicies/imagename/matcher"
)

func init() {
	compilers = append(compilers, newNameMatcher)
}

func newNameMatcher(policy *storage.Policy) (Matcher, error) {
	nameMatcher, err := imageNameMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if nameMatcher == nil {
		return nil, nil
	}

	return func(image *storage.Image) []*v1.Alert_Violation {
		return nameMatcher(image.GetName())
	}, nil
}
