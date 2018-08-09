package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	imageScanMatcher "github.com/stackrox/rox/pkg/compiledpolicies/imagescan/matcher"
)

func init() {
	compilers = append(compilers, newScanMatcher)
}

func newScanMatcher(policy *v1.Policy) (Matcher, error) {
	scanMatcher, err := imageScanMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if scanMatcher == nil {
		return nil, nil
	}

	return func(image *v1.Image) []*v1.Alert_Violation {
		return scanMatcher(image.GetScan())
	}, nil
}
