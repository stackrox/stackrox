package privilegeprocessor

import (
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// CompilePrivilegePolicy is a Privilege Policy that has been precompiled for matching deployments.
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

func init() {
	processors.PolicyCategoryCompiler[v1.Policy_Category_PRIVILEGES_CAPABILITIES] = NewCompiledPrivilegePolicy
}

// NewCompiledPrivilegePolicy returns a new compiledPrivilegePolicy.
func NewCompiledPrivilegePolicy(policy *v1.Policy) (compiledP processors.CompiledPolicy, err error) {
	if policy.GetPrivilegePolicy() == nil {
		return nil, fmt.Errorf("policy %s must contain privilege policy", policy.GetName())
	}
	privilegePolicy := policy.GetPrivilegePolicy()
	compiled := new(compiledPrivilegePolicy)
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

	return compiled, nil
}

func newCompiledSELinuxPolicy(policy *v1.PrivilegePolicy_SELinuxPolicy) (compiled *compiledSELinuxPolicy, err error) {
	if policy == nil {
		return
	}

	compiled = new(compiledSELinuxPolicy)
	compiled.User, err = processors.CompileStringRegex(policy.GetUser())
	if err != nil {
		return nil, fmt.Errorf("user: %s", err)
	}

	compiled.Role, err = processors.CompileStringRegex(policy.GetRole())
	if err != nil {
		return nil, fmt.Errorf("role: %s", err)
	}

	compiled.Type, err = processors.CompileStringRegex(policy.GetType())
	if err != nil {
		return nil, fmt.Errorf("type: %s", err)
	}

	compiled.Level, err = processors.CompileStringRegex(policy.GetLevel())
	if err != nil {
		return nil, fmt.Errorf("level: %s", err)
	}
	return
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
