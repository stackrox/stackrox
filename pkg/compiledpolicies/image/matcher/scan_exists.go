package matcher

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newScanExistsMatcher)
}

func newScanExistsMatcher(policy *storage.Policy) (Matcher, error) {
	noScanExists := policy.GetFields().GetNoScanExists()
	if !noScanExists {
		return nil, nil
	}
	matcher := &scanExistsMatcherImpl{}
	return matcher.match, nil
}

type scanExistsMatcherImpl struct {
}

func (p *scanExistsMatcherImpl) match(image *storage.Image) []*storage.Alert_Violation {
	var violations []*storage.Alert_Violation
	if image.GetScan() == nil {
		violations = append(violations, &storage.Alert_Violation{
			Message: fmt.Sprintf("Image '%s' has not been scanned", image.GetName().GetFullName()),
		})
	}
	return violations
}
