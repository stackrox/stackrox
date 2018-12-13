package matcher

import (
	"github.com/stackrox/rox/generated/storage"
	imageScanMatcher "github.com/stackrox/rox/pkg/compiledpolicies/imagescan/matcher"
)

func init() {
	compilers = append(compilers, newScanMatcher)
}

func newScanMatcher(policy *storage.Policy) (Matcher, error) {
	scanMatcher, err := imageScanMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if scanMatcher == nil {
		return nil, nil
	}

	return func(image *storage.Image) []*storage.Alert_Violation {
		return scanMatcher(image.GetScan())
	}, nil
}
