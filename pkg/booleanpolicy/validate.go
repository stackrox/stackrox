package booleanpolicy

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

type validateConfiguration struct {
	// If set to true, env var policies are strictly validated such that
	// policies with a non-raw source checking for a value are marked as invalid.
	//
	// See ROX-5208 for details.
	validateEnvVarSourceRestrictions bool
}

// ValidateOption models an option for validation.
type ValidateOption func(*validateConfiguration)

// ValidateEnvVarSourceRestrictions enables validation of env-var source
// restrictions as described/requested in ROX-5208.
func ValidateEnvVarSourceRestrictions() ValidateOption {
	return func(c *validateConfiguration) {
		c.validateEnvVarSourceRestrictions = true
	}
}

// Validate validates the policy, to make sure it's a well-formed Boolean policy.
func Validate(p *storage.Policy, options ...ValidateOption) error {
	configuration := &validateConfiguration{}
	for _, option := range options {
		option(configuration)
	}

	errorList := errorhelpers.NewErrorList("policy validation")
	if p.GetPolicyVersion() != Version {
		errorList.AddStringf("invalid version for boolean policy (got %q)", p.GetPolicyVersion())
	}
	if p.GetName() == "" {
		errorList.AddString("no name specified")
	}
	for _, section := range p.GetPolicySections() {
		errorList.AddError(validatePolicySection(section, configuration))
	}
	return errorList.ToError()
}

// validatePolicySection validates the format of a policy section
func validatePolicySection(s *storage.PolicySection, configuration *validateConfiguration) error {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("validation of section %q", s.GetSectionName()))

	seenFields := set.NewStringSet()
	for _, g := range s.GetPolicyGroups() {
		m, err := findFieldMetadata(g.GetFieldName(), configuration)
		switch err {
		case nil:
			// All good, proceed
		case errNoSuchField:
			errorList.AddStringf("policy criteria name %q is invalid", g.GetFieldName())
			continue
		default:
			errorList.AddWrapf(err, "failed to resolve metadata for field %q", g.GetFieldName())
			continue
		}

		if len(g.GetValues()) == 0 {
			errorList.AddStringf("no values for field %q", g.GetFieldName())
		}
		if !seenFields.Add(g.GetFieldName()) {
			errorList.AddStringf("field name %q found in multiple groups", g.GetFieldName())
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
