package privilegeprocessor

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.New("detection/privilege_processor")
)

type matchFunc func(*v1.SecurityContext) ([]*v1.Alert_Violation, bool)

func (p *compiledPrivilegePolicy) Match(deployment *v1.Deployment, container *v1.Container) (output []*v1.Alert_Violation) {
	security := container.GetSecurityContext()
	if security == nil {
		return
	}

	matchFunctions := []matchFunc{
		p.matchPrivileged,
		p.matchAddCap,
		p.matchDropCap,
		p.SELinux.match,
	}

	var violations, vs []*v1.Alert_Violation
	var exists bool

	// Every sub-policy that exists must match and return violations for the policy to match.
	for _, f := range matchFunctions {
		if vs, exists = f(security); exists && len(vs) == 0 {
			return
		}
		violations = append(violations, vs...)
	}

	output = violations
	return
}

func (p *compiledPrivilegePolicy) matchPrivileged(security *v1.SecurityContext) (violations []*v1.Alert_Violation, exists bool) {
	if p.privileged == nil {
		return
	}

	if security.GetPrivileged() != *p.privileged {
		return
	}

	exists = true
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
			Message: fmt.Sprintf("Container with drop capabilities %+v did not contain all configured drop capabilities %+v", security.GetDropCapabilities(), p.Original.GetPrivilegePolicy().GetDropCapabilities()),
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
			Message: fmt.Sprintf("Container with add capabilities %+v matches policy %+v", security.GetAddCapabilities(), p.Original.GetPrivilegePolicy().GetAddCapabilities()),
		})
	}

	return
}

func (p *compiledSELinuxPolicy) match(security *v1.SecurityContext) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}

	exists = true
	selinux := security.GetSelinux()
	if selinux == nil {
		return
	}

	if p.User != nil && !p.User.MatchString(selinux.GetUser()) {
		return
	}
	if p.Role != nil && !p.Role.MatchString(selinux.GetRole()) {
		return
	}
	if p.Type != nil && !p.Type.MatchString(selinux.GetType()) {
		return
	}
	if p.Level != nil && !p.Level.MatchString(selinux.GetLevel()) {
		return
	}

	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("SELinux %+v matched configured policy %s", selinux, p),
	})

	return
}
