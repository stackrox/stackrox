package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newVersionMatcher)
}

func newVersionMatcher(policy *v1.Policy) (Matcher, error) {
	versionPolicy := policy.GetFields().GetComponent().GetVersion()
	if versionPolicy == "" {
		return nil, nil
	}

	versionRegex, err := utils.CompileStringRegex(versionPolicy)
	if err != nil {
		return nil, err
	}
	matcher := &versionMatcherImpl{versionRegex}
	return matcher.match, nil
}

type versionMatcherImpl struct {
	versionRegex *regexp.Regexp
}

func (p *versionMatcherImpl) match(component *v1.ImageScanComponent) []*v1.Alert_Violation {
	if p.versionRegex.MatchString(component.GetVersion()) {
		return append(([]*v1.Alert_Violation)(nil), &v1.Alert_Violation{
			Message: fmt.Sprintf("Component '%v:%v' matches %s", component.GetName(), component.GetVersion(), p.versionRegex),
		})
	}
	return nil
}
