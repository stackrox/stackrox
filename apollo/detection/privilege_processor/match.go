package privilegeprocessor

import (
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.New("detection/privilege_processor")
)

type compiledPrivilegePolicy struct {
	Original *v1.Policy

	privileged *bool
	SELinux    *compiledSELinuxPolicy

	// container must contain all configured drop caps, otherwise alert.
	dropCap map[string]struct{}
	// alert if container contains all configured add caps.
	addCap map[string]struct{}
}

type compiledSELinuxPolicy struct {
	User  *regexp.Regexp
	Role  *regexp.Regexp
	Type  *regexp.Regexp
	Level *regexp.Regexp
}

func newCompiledPrivilegePolicy(policy *v1.Policy) (compiled *compiledPrivilegePolicy, err error) {
	if policy.GetPrivilegePolicy() == nil {
		return nil, fmt.Errorf("policy %s must contain privilege policy", policy.GetName())
	}
	privilegePolicy := policy.GetPrivilegePolicy()
	compiled = new(compiledPrivilegePolicy)
	compiled.Original = policy

	if privilegePolicy.GetSetPrivileged() != nil {
		priv := privilegePolicy.GetPrivileged()
		compiled.privileged = &priv
	}
	compiled.SELinux, err = newCompiledSELinuxPolicy(privilegePolicy.GetSelinux())
	if err != nil {
		return nil, fmt.Errorf("SELinux: %s", err)
	}

	compiled.dropCap = make(map[string]struct{})
	for _, cap := range privilegePolicy.GetDropCapabilities() {
		compiled.dropCap[strings.ToUpper(cap)] = struct{}{}
	}

	compiled.addCap = make(map[string]struct{})
	for _, cap := range privilegePolicy.GetAddCapabilities() {
		compiled.addCap[strings.ToUpper(cap)] = struct{}{}
	}

	return
}

func newCompiledSELinuxPolicy(policy *v1.PrivilegePolicy_SELinuxPolicy) (compiled *compiledSELinuxPolicy, err error) {
	if policy == nil {
		return
	}

	compiled = new(compiledSELinuxPolicy)
	compiled.User, err = compileStringRegex(policy.GetUser())
	if err != nil {
		return nil, fmt.Errorf("user: %s", err)
	}

	compiled.Role, err = compileStringRegex(policy.GetRole())
	if err != nil {
		return nil, fmt.Errorf("role: %s", err)
	}

	compiled.Type, err = compileStringRegex(policy.GetType())
	if err != nil {
		return nil, fmt.Errorf("type: %s", err)
	}

	compiled.Level, err = compileStringRegex(policy.GetLevel())
	if err != nil {
		return nil, fmt.Errorf("level: %s", err)
	}
	return
}

func compileStringRegex(regex string) (*regexp.Regexp, error) {
	if regex == "" {
		return nil, nil
	}
	return regexp.Compile(regex)
}

func (p *compiledSELinuxPolicy) String() string {
	var fields []string
	if p.User != nil {
		fields = append(fields, fmt.Sprintf("user=%v", p.User))
	}
	if p.Role != nil {
		fields = append(fields, fmt.Sprintf("role=%v", p.Role))
	}
	if p.Type != nil {
		fields = append(fields, fmt.Sprintf("type=%v", p.Type))
	}
	if p.Level != nil {
		fields = append(fields, fmt.Sprintf("level=%v", p.Level))
	}
	return strings.Join(fields, ", ")
}

type matchFunc func(*v1.SecurityContext) ([]*v1.Alert_Violation, bool)

// Match checks whether a policy matches a given deployment.
// Each container is considered independently.
func (p *compiledPrivilegePolicy) match(deployment *v1.Deployment) (violations []*v1.Alert_Violation) {
	for _, c := range deployment.GetContainers() {
		violations = append(violations, p.matchContainer(c.GetSecurityContext())...)
	}

	return
}

func (p *compiledPrivilegePolicy) matchContainer(security *v1.SecurityContext) (output []*v1.Alert_Violation) {
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
