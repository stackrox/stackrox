package matcher

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	compilers = append(compilers, newScanExistsMatcher)
}

func newScanExistsMatcher(policy *v1.Policy) (Matcher, error) {
	hasCanExists := policy.GetFields().GetSetScanExists()
	if hasCanExists == nil {
		return nil, nil
	}

	scanExists := policy.GetFields().GetScanExists()
	matcher := &scanExistsMatcherImpl{scanExists: &scanExists}
	return matcher.match, nil
}

type scanExistsMatcherImpl struct {
	scanExists *bool
}

func (p *scanExistsMatcherImpl) match(image *v1.Image) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	if *p.scanExists && image.GetScan() == nil {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Image '%s' has not been scanned", image.GetName().GetFullName()),
		})
	}
	return violations
}
