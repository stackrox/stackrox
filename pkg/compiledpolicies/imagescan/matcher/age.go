package matcher

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	compilers = append(compilers, newAgeMatcher)
}

func newAgeMatcher(policy *v1.Policy) (Matcher, error) {
	hasImageAge := policy.GetFields().GetSetScanAgeDays()
	if hasImageAge == nil {
		return nil, nil
	}

	scanAge := policy.GetFields().GetScanAgeDays()
	matcher := &ageMatcherImpl{scanAgeDays: &scanAge}
	return matcher.match, nil
}

type ageMatcherImpl struct {
	scanAgeDays *int64
}

func (p *ageMatcherImpl) match(scan *v1.ImageScan) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	deadline := time.Now().AddDate(0, 0, -int(*p.scanAgeDays))
	scanned := scan.GetScanTime()
	if scanned == nil {
		return nil
	}
	scannedTime, err := ptypes.TimestampFromProto(scanned)
	if err != nil {
		log.Error(err)
		return nil
	}
	if scannedTime.Before(deadline) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Scan Age '%v' is %0.2f days past the deadline", scannedTime, deadline.Sub(scannedTime).Hours()/24),
		})
	}
	return violations
}
