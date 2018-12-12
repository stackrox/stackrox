package matcher

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newPrivilegeMatcher)
}

func newPrivilegeMatcher(policy *storage.Policy) (Matcher, error) {
	fields := policy.GetFields()
	if fields.GetSetPrivileged() == nil {
		return nil, nil
	}

	privileged := fields.GetPrivileged()
	matcher := &privilegeMatcherImpl{privileged: &privileged}
	return matcher.match, nil
}

type privilegeMatcherImpl struct {
	privileged *bool
}

func (p *privilegeMatcherImpl) match(security *storage.SecurityContext) []*v1.Alert_Violation {
	if security == nil || security.GetPrivileged() != *p.privileged {
		return nil
	}

	var violations []*v1.Alert_Violation
	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Container privileged set to %t matched configured policy", security.GetPrivileged()),
	})
	return violations
}
