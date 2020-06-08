package booleanpolicy

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// Validate validates the policy, to make sure it's a well-formed Boolean policy.
func Validate(p *storage.Policy) error {
	errorList := errorhelpers.NewErrorList("policy validation")
	if p.GetPolicyVersion() != Version {
		errorList.AddStringf("invalid version for boolean policy (got %q)", p.GetPolicyVersion())
	}
	if p.GetName() == "" {
		errorList.AddString("no name specified")
	}
	for _, section := range p.GetPolicySections() {
		errorList.AddError(validatePolicySection(section))
	}
	return errorList.ToError()
}

// validatePolicySection validates the format of a policy section
func validatePolicySection(s *storage.PolicySection) error {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("validation of section %q", s.GetSectionName()))

	for _, g := range s.GetPolicyGroups() {
		m, ok := fieldsToQB[g.GetFieldName()]
		if !ok {
			errorList.AddStringf("policy criteria name %q is invalid", g.GetFieldName())
			continue
		}
		if len(g.GetValues()) == 0 {
			errorList.AddStringf("no values for field %q", g.GetFieldName())
		}
		if g.GetNegate() && m.negationForbidden {
			errorList.AddStringf("policy criteria %q cannot be negated", g.GetFieldName())
		}
		if len(g.GetValues()) > 1 && m.operatorsForbidden {
			errorList.AddStringf("policy criteria %q does not support more than one value %q", g.GetFieldName(), g.GetValues())
		}
		for idx, v := range g.GetValues() {
			if !m.valueRegex.MatchString(v.GetValue()) {
				errorList.AddStringf("policy criteria %q has invalid value[%d]=%q must match regex %q", g.GetFieldName(), idx, v.GetValue(), m.valueRegex)
			}
		}
	}
	return errorList.ToError()
}
