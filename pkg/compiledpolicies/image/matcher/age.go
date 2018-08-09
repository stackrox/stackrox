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
	hasImageAge := policy.GetFields().GetSetImageAgeDays()
	if hasImageAge == nil {
		return nil, nil
	}

	imageAge := policy.GetFields().GetImageAgeDays()
	matcher := ageMatcherImpl{imageAgeDays: &imageAge}
	return matcher.match, nil
}

type ageMatcherImpl struct {
	imageAgeDays *int64
}

func (p *ageMatcherImpl) match(image *v1.Image) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	deadline := time.Now().AddDate(0, 0, -int(*p.imageAgeDays))
	created := image.GetMetadata().GetCreated()
	if created == nil {
		return nil
	}

	createdTime, err := ptypes.TimestampFromProto(created)
	if err != nil {
		log.Error(err) // Log just in case, though in reality this should not occur
		return nil
	}

	if createdTime.Before(deadline) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Image Age '%v' is %0.2f days past the deadline", createdTime, deadline.Sub(createdTime).Hours()/24),
		})
	}
	return violations
}
