package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newTypeMatcher)
}

func newTypeMatcher(policy *v1.Policy) (Matcher, error) {
	vtype := policy.GetFields().GetVolumePolicy().GetType()
	if vtype == "" {
		return nil, nil
	}

	typeRegex, err := utils.CompileStringRegex(vtype)
	if err != nil {
		return nil, err
	}
	matcher := &typeMatcherImpl{typeRegex}
	return matcher.match, nil
}

type typeMatcherImpl struct {
	vtypeRegex *regexp.Regexp
}

func (p *typeMatcherImpl) match(volume *storage.Volume) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	if p.vtypeRegex.MatchString(volume.GetType()) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Volume type matched: %s", p.vtypeRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
