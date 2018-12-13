package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newVersionMatcher)
}

func newVersionMatcher(policy *storage.Policy) (Matcher, error) {
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

func (p *versionMatcherImpl) match(component *storage.ImageScanComponent) []*storage.Alert_Violation {
	if p.versionRegex.MatchString(component.GetVersion()) {
		return append(([]*storage.Alert_Violation)(nil), &storage.Alert_Violation{
			Message: fmt.Sprintf("Component '%v:%v' matches %s", component.GetName(), component.GetVersion(), p.versionRegex),
		})
	}
	return nil
}
