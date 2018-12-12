package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newNameMatcher)
}

func newNameMatcher(policy *storage.Policy) (Matcher, error) {
	namePolicy := policy.GetFields().GetComponent().GetName()
	if namePolicy == "" {
		return nil, nil
	}

	nameRegex, err := utils.CompileStringRegex(namePolicy)
	if err != nil {
		return nil, err
	}
	matcher := &nameMatcherImpl{nameRegex}
	return matcher.match, nil
}

type nameMatcherImpl struct {
	nameRegex *regexp.Regexp
}

func (p *nameMatcherImpl) match(component *storage.ImageScanComponent) []*v1.Alert_Violation {
	if p.nameRegex.MatchString(component.GetName()) {
		return append(([]*v1.Alert_Violation)(nil), &v1.Alert_Violation{
			Message: fmt.Sprintf("Component '%v:%v' matches %s", component.GetName(), component.GetVersion(), p.nameRegex),
		})
	}
	return nil
}
