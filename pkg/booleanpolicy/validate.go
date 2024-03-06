package booleanpolicy

import (
	"errors"
	"fmt"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

type validateConfiguration struct {
	// If set to true, env var policies are strictly validated such that
	// policies with a non-raw source checking for a value are marked as invalid.
	//
	// See ROX-5208 for details.
	validateEnvVarSourceRestrictions bool
	sourceIsAuditLogEvents           bool
	disallowFromInDockerfileLine     bool
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

// ValidateNoFromInDockerfileLine disallows FROM in the Dockerfile line.
func ValidateNoFromInDockerfileLine() ValidateOption {
	return func(c *validateConfiguration) {
		c.disallowFromInDockerfileLine = true
	}
}

// ValidateSourceIsAuditLogEvents enables validation audit log event based criteria
func ValidateSourceIsAuditLogEvents() ValidateOption {
	return func(c *validateConfiguration) {
		c.sourceIsAuditLogEvents = true
	}
}

// Validate validates the policy, to make sure it's a well-formed Boolean policy.
func Validate(p *storage.Policy, options ...ValidateOption) error {
	if p.GetEventSource() == storage.EventSource_AUDIT_LOG_EVENT {
		options = append(options, ValidateSourceIsAuditLogEvents())
	}
	configuration := &validateConfiguration{}
	for _, option := range options {
		option(configuration)
	}

	var validationErrs error
	if err := validateBooleanPolicyVersion(p); err != nil {
		validationErrs = errors.Join(validationErrs, err)
	}
	if p.GetName() == "" {
		validationErrs = errors.Join(validationErrs, errors.New("no name specified"))
	}

	for _, section := range p.GetPolicySections() {
		validationErrs = errors.Join(validationErrs, validatePolicySection(section, configuration, p.GetEventSource()))
	}

	// Special case for ImageSignatureVerifiedBy policy for which we don't allow
	// AND operator due to the UI limitations.
	for _, ps := range p.PolicySections {
		for _, pg := range ps.PolicyGroups {
			if pg.FieldName == fieldnames.ImageSignatureVerifiedBy && pg.BooleanOperator == storage.BooleanOperator_AND {
				validationErrs = errors.Join(validationErrs,
					fmt.Errorf("operator AND is not allowed for field %q", fieldnames.ImageSignatureVerifiedBy))
			}
		}
	}

	return pkgErrors.Wrap(validationErrs, "policy validation")
}

func validateBooleanPolicyVersion(policy *storage.Policy) error {
	ver, err := policyversion.FromString(policy.GetPolicyVersion())
	if err != nil {
		return errors.New("policy has invalid version")
	}

	// As of 70.0 we only support the latest version (1.1). This may in the future be expanded to support more, but for
	// now it's enough to just check it matches current version
	if !policyversion.IsCurrentVersion(ver) {
		return errors.New("only policy with version 1.1 is supported")
	}
	return nil
}

// validatePolicySection validates the format of a policy section
func validatePolicySection(s *storage.PolicySection, configuration *validateConfiguration, eventSource storage.EventSource) error {
	var validationErrs error
	seenFields := set.NewStringSet()
	for _, g := range s.GetPolicyGroups() {
		m, err := FieldMetadataSingleton().findFieldMetadata(g.GetFieldName(), configuration)
		switch err {
		case nil:
			// All good, proceed
		case errNoSuchField:
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("policy criteria name %q is invalid", g.GetFieldName()))
			continue
		default:
			validationErrs = errors.Join(validationErrs,
				pkgErrors.Wrapf(err, "failed to resolve metadata for field %q", g.GetFieldName()))
			continue
		}

		if len(g.GetValues()) == 0 {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("no values for field %q", g.GetFieldName()))
		}
		if !seenFields.Add(g.GetFieldName()) {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("field name %q found in multiple groups", g.GetFieldName()))
		}
		if g.GetNegate() && m.negationForbidden {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("policy criteria %q cannot be negated", g.GetFieldName()))
		}
		if len(g.GetValues()) > 1 && m.operatorsForbidden {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("policy criteria %q does not support more than one value %q", g.GetFieldName(), g.GetValues()))
		}
		for idx, v := range g.GetValues() {
			if !m.valueRegex(configuration).MatchString(v.GetValue()) {
				validationErrs = errors.Join(validationErrs,
					errox.InvalidArgs.Newf("policy criteria %q has invalid value[%d]=%q must match regex %q",
						g.GetFieldName(), idx, v.GetValue(), m.valueRegex(configuration).String()))
			}
		}
	}

	if eventSource == storage.EventSource_AUDIT_LOG_EVENT {
		// For Audit Log source based policies, both the k8s resource and verb must be provided.
		if !seenFields.Contains(fieldnames.KubeResource) {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("policies with audit log event source must have the %q criteria", fieldnames.KubeResource))
		}
		if !seenFields.Contains(fieldnames.KubeAPIVerb) {
			validationErrs = errors.Join(validationErrs,
				errox.InvalidArgs.Newf("policies with audit log event source must have the %q criteria", fieldnames.KubeAPIVerb))
		}
	}

	if eventSource == storage.EventSource_DEPLOYMENT_EVENT {
		if seenFields.Contains(fieldnames.KubeUserName) || seenFields.Contains(fieldnames.KubeUserGroups) {
			if !seenFields.Contains(fieldnames.KubeResource) {
				validationErrs = errors.Join(validationErrs,
					errox.InvalidArgs.New("kubernetes events policy must have the `Kubernetes Action` criteria"))
			}
		}
	}
	return pkgErrors.Wrapf(validationErrs, "validation of section %q", s.GetSectionName())
}
