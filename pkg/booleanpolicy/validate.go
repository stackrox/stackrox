package booleanpolicy

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// fieldDependencies defines the dependencies between fields in a policy
	// section. For each key field name in the map, the values is the set of
	// one OR more fields that must also exist to pass the validation
	//
	// Note that the Key -> [Values] dependency exists, but the reverse
	// is not valid. The values can exist on their own in a policy without
	// requiring the key to also exist.
	fieldDependencies = map[string]set.StringSet{
		fieldnames.FileOperation: set.NewStringSet(
			fieldnames.ActualPath,
			fieldnames.EffectivePath,
		),
		fieldnames.KubeUserName: set.NewStringSet(
			fieldnames.KubeResource,
		),
		fieldnames.KubeUserGroups: set.NewStringSet(
			fieldnames.KubeResource,
		),
	}

	// eventSourceRequirements defines the minimum required fields for a
	// given event source.
	eventSourceRequirements = map[storage.EventSource]set.StringSet{
		storage.EventSource_AUDIT_LOG_EVENT: set.NewStringSet(
			fieldnames.KubeResource,
			fieldnames.KubeAPIVerb,
		),
		// FileAccess fields are currently the only ones supported for
		// node events. In the future, when more node events are supported,
		// this constraint can be relaxed.
		storage.EventSource_NODE_EVENT: set.NewStringSet(
			fieldnames.ActualPath,
		),
	}
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

	errorList := errorhelpers.NewErrorList("policy validation")
	if err := validateBooleanPolicyVersion(p); err != nil {
		errorList.AddErrors(err)
	}
	if p.GetName() == "" {
		errorList.AddString("no name specified")
	}

	for _, section := range p.GetPolicySections() {
		errorList.AddError(validatePolicySection(section, configuration, p.GetEventSource()))
	}

	// Special case for ImageSignatureVerifiedBy policy for which we don't allow
	// AND operator due to the UI limitations.
	for _, ps := range p.GetPolicySections() {
		for _, pg := range ps.GetPolicyGroups() {
			if pg.GetFieldName() == fieldnames.ImageSignatureVerifiedBy && pg.GetBooleanOperator() == storage.BooleanOperator_AND {
				errorList.AddStringf("operator AND is not allowed for field %q", fieldnames.ImageSignatureVerifiedBy)
			}
		}
	}

	return errorList.ToError()
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
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("validation of section %q", s.GetSectionName()))

	seenFields := set.NewStringSet()
	metadata := FieldMetadataSingleton()
	for _, g := range s.GetPolicyGroups() {
		m, err := metadata.findFieldMetadata(g.GetFieldName(), configuration)
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

		// For fields that apply to an event source, validate that they match
		// the policy's event source.
		if !m.IsNotApplicableEventSource() && !m.IsFromEventSource(eventSource) {
			errorList.AddStringf("%q is not supported for event source %q", g.GetFieldName(), eventSource)
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
			if !m.valueRegex(configuration).MatchString(v.GetValue()) {
				errorList.AddStringf("policy criteria %q has invalid value[%d]=%q must match regex %q", g.GetFieldName(), idx, v.GetValue(), m.valueRegex(configuration).String())
			}
		}
	}

	if err := validateEventSourceRequirements(s, &seenFields, eventSource); err != nil {
		errorList.AddError(err)
	}

	if err := validateFieldDependencies(s, &seenFields); err != nil {
		errorList.AddError(err)
	}

	return errorList.ToError()
}

// validateFieldDependencies validates a policy section with respect to field dependencies,
// as outlined in the fieldDependencies map.
func validateFieldDependencies(s *storage.PolicySection, seenFields *set.StringSet) error {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("validating field dependencies for %q", s.GetSectionName()))

	for field, dependencies := range fieldDependencies {
		if seenFields.Contains(field) && !slices.ContainsFunc(dependencies.AsSlice(), seenFields.Contains) {
			errorList.AddStringf("policy sections with %s must also contain %s", field, strings.Join(dependencies.AsSlice(), " or "))
		}
	}

	return errorList.ToError()
}

// validateEventSourceRequirements validates a policy section with respect to
// required fields as outlined in the eventSourceRequirements map.
func validateEventSourceRequirements(s *storage.PolicySection, seenFields *set.StringSet, eventSource storage.EventSource) error {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("validating event source requirements for %s", s.GetSectionName()))

	for es, requiredFields := range eventSourceRequirements {
		if eventSource != es {
			continue
		}

		for required := range requiredFields {
			if !seenFields.Contains(required) {
				errorList.AddStringf("%q policies require field %q", eventSource, required)
			}
		}
	}

	return errorList.ToError()
}
