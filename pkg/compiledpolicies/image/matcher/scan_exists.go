package matcher

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	compilers = append(compilers, newScanExistsMatcher)
}

func newScanExistsMatcher(policy *v1.Policy) (Matcher, error) {
	noScanExists := policy.GetFields().GetNoScanExists()
	if !noScanExists {
		return nil, nil
	}
	matcher := &scanExistsMatcherImpl{}
	return matcher.match, nil
}

type scanExistsMatcherImpl struct {
}

func (p *scanExistsMatcherImpl) match(image *v1.Image) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	if image.GetScan() == nil {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Image '%s' has not been scanned", image.GetName().GetFullName()),
		})
	}
	return violations
}
