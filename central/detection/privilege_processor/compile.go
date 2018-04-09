package privilegeprocessor

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// CompilePrivilegePolicy is a Privilege Policy that has been precompiled for matching deployments.
type compiledPrivilegePolicy struct {
	Original *v1.Policy

	privileged *bool

	// container must contain all configured drop caps, otherwise alert.
	dropCap map[string]struct{}
	// alert if container contains all configured add caps.
	addCap map[string]struct{}
}

func init() {
	processors.PolicySegmentCompilers = append(processors.PolicySegmentCompilers, NewCompiledPrivilegePolicy)
}

// NewCompiledPrivilegePolicy returns a new compiledPrivilegePolicy.
func NewCompiledPrivilegePolicy(policy *v1.Policy) (compiledP processors.CompiledPolicy, exist bool, err error) {
	if policy.GetPrivilegePolicy() == nil {
		return
	}

	exist = true
	privilegePolicy := policy.GetPrivilegePolicy()
	compiled := new(compiledPrivilegePolicy)
	compiled.Original = policy

	if privilegePolicy.GetSetPrivileged() != nil {
		priv := privilegePolicy.GetPrivileged()
		compiled.privileged = &priv
	}
	compiled.dropCap = make(map[string]struct{})
	for _, cap := range privilegePolicy.GetDropCapabilities() {
		compiled.dropCap[strings.ToUpper(cap)] = struct{}{}
	}

	compiled.addCap = make(map[string]struct{})
	for _, cap := range privilegePolicy.GetAddCapabilities() {
		compiled.addCap[strings.ToUpper(cap)] = struct{}{}
	}

	return compiled, exist, nil
}
