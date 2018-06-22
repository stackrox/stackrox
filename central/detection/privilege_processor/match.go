package privilegeprocessor

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

type matchFunc func(*v1.SecurityContext) ([]*v1.Alert_Violation, bool)

func (p *compiledPrivilegePolicy) MatchDeployment(*v1.Deployment) ([]*v1.Alert_Violation, bool) {
	return nil, false
}

func (p *compiledPrivilegePolicy) MatchContainer(container *v1.Container) ([]*v1.Alert_Violation, bool) {
	security := container.GetSecurityContext()
	if security == nil {
		return nil, false
	}

	matchFunctions := []matchFunc{
		p.matchPrivileged,
		p.matchAddCap,
		p.matchDropCap,
	}

	var violations []*v1.Alert_Violation
	var exists bool

	// Every sub-policy that exists must match and return violations for the policy to match.
	for _, f := range matchFunctions {
		vs, valid := f(security)
		if valid && len(vs) == 0 {
			return nil, true
		} else if valid {
			exists = true
		}
		violations = append(violations, vs...)
	}
	return violations, exists
}

func (p *compiledPrivilegePolicy) matchPrivileged(security *v1.SecurityContext) (violations []*v1.Alert_Violation, exists bool) {
	if p.privileged == nil {
		return
	}

	exists = true
	if security.GetPrivileged() != *p.privileged {
		return
	}

	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Container privileged set to %t matched configured policy", security.GetPrivileged()),
	})

	return
}

func (p *compiledPrivilegePolicy) matchDropCap(security *v1.SecurityContext) (violations []*v1.Alert_Violation, exists bool) {
	if len(p.dropCap) == 0 {
		return
	}

	exists = true
	matchedCap := 0
	// assuming no duplicates.
	for _, cap := range security.GetDropCapabilities() {
		if _, ok := p.dropCap[strings.ToUpper(cap)]; ok {
			matchedCap++
		}
	}

	if matchedCap < len(p.dropCap) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Container with drop capabilities %+v did not contain all configured drop capabilities %+v", security.GetDropCapabilities(), p.Original.GetFields().GetDropCapabilities()),
		})
	}

	return
}

func (p *compiledPrivilegePolicy) matchAddCap(security *v1.SecurityContext) (violations []*v1.Alert_Violation, exists bool) {
	if len(p.addCap) == 0 {
		return
	}

	exists = true
	matchedCap := 0
	for _, cap := range security.GetAddCapabilities() {
		if _, ok := p.addCap[strings.ToUpper(cap)]; ok {
			matchedCap++
		}
	}

	if len(p.addCap) == matchedCap {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Container with add capabilities %+v matches policy %+v", security.GetAddCapabilities(), p.Original.GetFields().GetAddCapabilities()),
		})
	}

	return
}
