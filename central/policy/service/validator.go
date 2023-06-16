package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/scopecomp"
	"github.com/stackrox/rox/pkg/set"
)

var (
	nameValidator = &regexpAndDesc{
		r:    regexp.MustCompile(`^[^\n\r\$]{5,128}$`),
		desc: "policy must have a name between 5 and 128 characters long with no new lines or dollar signs",
	}
	descriptionValidator = &regexpAndDesc{
		r:    regexp.MustCompile(`^[^\$]{0,800}$`),
		desc: "description, when present, should be of sentence form, and not contain more than 800 characters",
	}
)

type regexpAndDesc struct {
	r    *regexp.Regexp
	desc string
}

func (r *regexpAndDesc) Validate(s string) error {
	if !r.r.MatchString(s) {
		return errors.New(r.desc)
	}
	return nil
}

func newPolicyValidator(notifierStorage notifierDataStore.DataStore) *policyValidator {
	pv := &policyValidator{
		notifierStorage:            notifierStorage,
		nonEnforceablePolicyFields: make(map[string]struct{}),
	}
	pv.nonEnforceablePolicyFields[augmentedobjs.HasIngressPolicyCustomTag] = struct{}{}
	pv.nonEnforceablePolicyFields[augmentedobjs.HasEgressPolicyCustomTag] = struct{}{}
	return pv
}

type validationFunc func(*storage.Policy) error

// policyValidator validates the incoming policy.
type policyValidator struct {
	notifierStorage            notifierDataStore.DataStore
	nonEnforceablePolicyFields map[string]struct{}
}

func (s *policyValidator) validate(ctx context.Context, policy *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	s.removeEnforcementsForMissingLifecycles(policy)

	additionalValidators := []validationFunc{
		func(policy *storage.Policy) error {
			return s.validateNotifiers(ctx, policy)
		},
	}
	return s.internalValidate(policy, additionalValidators, options...)
}

// validateImport does not validate notifiers.  Invalid notifiers will be stripped out.
func (s *policyValidator) validateImport(policy *storage.Policy) error {
	return s.internalValidate(policy, nil)
}

// internalValidate validates policy.
//
// additionalValidators should be used for 'shallow' extra validations, e.g., checking if a name satisfies a certain pattern.
// options are propagated to the booleanpolicy validation engine.
func (s *policyValidator) internalValidate(policy *storage.Policy, additionalValidators []validationFunc, options ...booleanpolicy.ValidateOption) error {
	s.removeEnforcementsForMissingLifecycles(policy)

	errorList := errorhelpers.NewErrorList("policy invalid")
	errorList.AddError(s.validateVersion(policy))
	errorList.AddError(s.validateName(policy))
	errorList.AddError(s.validateDescription(policy))
	errorList.AddError(s.validateCompilableForLifecycle(policy, options...))
	errorList.AddError(s.validateSeverity(policy))
	errorList.AddError(s.validateCategories(policy))
	errorList.AddError(s.validateScopes(policy))
	errorList.AddError(s.validateExclusions(policy))
	errorList.AddError(s.validateCapabilities(policy))
	errorList.AddError(s.validateEventSource(policy))
	errorList.AddError(s.validateEnforcement(policy))

	for _, validator := range additionalValidators {
		errorList.AddError(validator(policy))
	}
	return errorList.ToError()
}

func (s *policyValidator) validateVersion(policy *storage.Policy) error {
	ver, err := policyversion.FromString(policy.GetPolicyVersion())
	if err != nil {
		return errors.New("policy has invalid version")
	}

	// As of 70.0 we only support the latest version (1.1).
	if !policyversion.IsSupportedVersion(ver) {
		return errors.Errorf("policy version %s is not supported", ver.String())
	}
	return nil
}

func (s *policyValidator) validateName(policy *storage.Policy) error {
	policy.Name = strings.TrimSpace(policy.Name)
	return nameValidator.Validate(policy.GetName())
}

func (s *policyValidator) validateDescription(policy *storage.Policy) error {
	return descriptionValidator.Validate(policy.GetDescription())
}

func (s *policyValidator) validateCompilableForLifecycle(policy *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	if len(policy.GetLifecycleStages()) == 0 {
		return errors.New("a policy must apply to at least one lifecycle stage")
	}

	errorList := errorhelpers.NewErrorList("error validating lifecycle stage")
	if policies.AppliesAtBuildTime(policy) {
		errorList.AddError(s.compilesForBuildTime(policy, options...))
	}
	if policies.AppliesAtDeployTime(policy) {
		errorList.AddError(s.compilesForDeployTime(policy, options...))
	}
	if policies.AppliesAtRunTime(policy) {
		errorList.AddError(s.compilesForRunTime(policy, options...))
	}
	return errorList.ToError()
}

func (s *policyValidator) removeEnforcementsForMissingLifecycles(policy *storage.Policy) {
	if !policies.AppliesAtBuildTime(policy) {
		removeEnforcementForLifecycle(policy, storage.LifecycleStage_BUILD)
	}
	if !policies.AppliesAtDeployTime(policy) {
		removeEnforcementForLifecycle(policy, storage.LifecycleStage_DEPLOY)
	}
	if !policies.AppliesAtRunTime(policy) {
		removeEnforcementForLifecycle(policy, storage.LifecycleStage_RUNTIME)
	}
}

func (s *policyValidator) validateEventSource(policy *storage.Policy) error {
	if policies.AppliesAtRunTime(policy) && policy.GetEventSource() == storage.EventSource_NOT_APPLICABLE {
		return errors.New("event source must be deployment or audit event for runtime policies")
	}

	if (policies.AppliesAtBuildTime(policy) || policies.AppliesAtDeployTime(policy)) &&
		policy.GetEventSource() != storage.EventSource_NOT_APPLICABLE {
		return errors.New("event source must not be set for build or deploy time policies")
	}

	if s.isAuditEventPolicy(policy) {
		if len(policy.GetEnforcementActions()) != 0 {
			return errors.New("enforcement actions are not applicable for runtime policies with audit log as the event source")
		}

		for _, s := range policy.GetScope() {
			if err := validateNoLabelsInScopeForAuditEvent(s, "restrict to scope"); err != nil {
				return err
			}
		}
		for _, e := range policy.GetExclusions() {
			if e.GetDeployment() != nil {
				if e.GetDeployment().GetName() != "" {
					return errors.New("deployment level exclusion is not applicable runtime policies with audit log as the event source")
				}
				if err := validateNoLabelsInScopeForAuditEvent(e.GetDeployment().GetScope(), "exclude by scope"); err != nil {
					return err
				}
			}
		}
	}
	// TODO(@khushboo): ROX-7252: Modify this validation once migration to account for new policy field event source is in
	return nil
}

func validateNoLabelsInScopeForAuditEvent(scope *storage.Scope, context string) error {
	if scope.GetLabel() != nil {
		return errors.Errorf("labels in `%s` section are not permitted for audit log events based policies", context)
	}
	return nil
}

func (s *policyValidator) validateSeverity(policy *storage.Policy) error {
	if policy.GetSeverity() == storage.Severity_UNSET_SEVERITY {
		return errors.New("a policy must have a severity")
	}
	return nil
}

func (s *policyValidator) getCaps(policy *storage.Policy, capsTypes string) []*storage.PolicyValue {
	capsValues := make([]*storage.PolicyValue, 0)
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if group.GetFieldName() == capsTypes {
				capsValues = append(capsValues, group.Values...)
			}
		}
	}
	return capsValues
}

func (s *policyValidator) validateCapabilities(policy *storage.Policy) error {
	values := set.NewSet[string]()
	for _, s := range s.getCaps(policy, fieldnames.AddCaps) {
		values.Add(s.GetValue())
	}
	var duplicates []string
	for _, s := range s.getCaps(policy, fieldnames.DropCaps) {
		// We use `Remove` to ensure that each duplicate value is reported only once.
		if val := s.GetValue(); values.Remove(val) {
			duplicates = append(duplicates, val)
		}
	}
	if len(duplicates) != 0 {
		return fmt.Errorf("Capabilities '%s' cannot be included in both add and drop", strings.Join(duplicates, ","))
	}
	return nil
}

func (s *policyValidator) validateCategories(policy *storage.Policy) error {
	if len(policy.GetCategories()) == 0 {
		return errors.New("a policy must have at least one category")
	}
	categorySet := make(map[string]struct{})
	for _, c := range policy.GetCategories() {
		categorySet[c] = struct{}{}
	}
	if len(categorySet) != len(policy.GetCategories()) {
		return errors.New("a policy cannot contain duplicate categories")
	}
	return nil
}

func (s *policyValidator) validateNotifiers(ctx context.Context, policy *storage.Policy) error {
	for _, n := range policy.GetNotifiers() {
		_, exists, err := s.notifierStorage.GetNotifier(ctx, n)
		if err != nil {
			return fmt.Errorf("error checking if notifier %s is valid", n)
		}
		if !exists {
			return fmt.Errorf("notifier %s does not exist", n)
		}
	}
	return nil
}

func (s *policyValidator) validateScopes(policy *storage.Policy) error {
	for _, scope := range policy.GetScope() {
		if err := s.validateScope(scope); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateExclusions(policy *storage.Policy) error {
	for _, exclusion := range policy.GetExclusions() {
		if err := s.validateExclusion(policy, exclusion); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateExclusion(policy *storage.Policy, exclusion *storage.Exclusion) error {
	if exclusion.GetDeployment() == nil && exclusion.GetImage() == nil {
		return errors.New("all excluded scopes must have some criteria to match on")
	}
	if exclusion.GetDeployment() != nil {
		if !policies.AppliesAtDeployTime(policy) && !policies.AppliesAtRunTime(policy) {
			return errors.New("excluding a deployment is only valid during the DEPLOY and RUNTIME lifecycles")
		}
		if err := s.validateDeploymentExclusion(exclusion); err != nil {
			return err
		}
	}
	if exclusion.GetImage() != nil {
		if !policies.AppliesAtBuildTime(policy) {
			return errors.New("excluding an image is only valid during the BUILD lifecycle")
		}
		if exclusion.GetImage().GetName() == "" {
			return errors.New("image excluded scope must have nonempty name")
		}
	}
	return nil
}

func (s *policyValidator) validateDeploymentExclusion(exclusion *storage.Exclusion) error {
	deployment := exclusion.GetDeployment()
	if deployment.GetScope() == nil && deployment.GetName() == "" {
		return errors.New("at least one field of deployment exclusion scope must be defined")
	}
	if deployment.GetScope() != nil {
		if err := s.validateScope(deployment.GetScope()); err != nil {
			return errors.Wrap(err, "deployment exclusion scope is invalid")
		}
	}
	return nil
}

func (s *policyValidator) validateScope(scope *storage.Scope) error {
	if scope.GetCluster() == "" && scope.GetNamespace() == "" && scope.GetLabel() == nil {
		return errors.New("scope must have at least one field populated")
	}
	if _, err := scopecomp.CompileScope(scope); err != nil {
		return errors.Wrap(err, "could not compile scope")
	}
	return nil
}

func (s *policyValidator) compilesForBuildTime(policy *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	_, err := booleanpolicy.BuildImageMatcher(policy, options...)
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for build time")
	}
	return nil
}

func (s *policyValidator) compilesForDeployTime(policy *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	_, err := booleanpolicy.BuildDeploymentMatcher(policy, options...)
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for deploy time")
	}
	if booleanpolicy.ContainsRuntimeFields(policy) {
		return errors.New("deploy time policy cannot contain runtime criteria")
	}
	return nil
}

func (s *policyValidator) compilesForRunTime(policy *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	// Runtime policies must contain one category of runtime criteria, but can have deploy time criteria as well
	if !booleanpolicy.ContainsRuntimeFields(policy) {
		return errors.New("A runtime policy must contain at least one policy criterion from process, network flow, audit log events, or Kubernetes events criteria categories")
	}

	if !booleanpolicy.ContainsDiscreteRuntimeFieldCategorySections(policy) {
		return errors.New("A runtime policy section must contain only one criterion from process, network flow, audit log events, or Kubernetes events criteria categories")
	}

	var err error
	if s.isAuditEventPolicy(policy) {
		_, err = booleanpolicy.BuildAuditLogEventMatcher(policy, booleanpolicy.ValidateSourceIsAuditLogEvents())
	} else {
		// build a deployment matcher to check for all runtime fields that are evaluated against a deployment
		_, err = booleanpolicy.BuildDeploymentMatcher(policy, options...)
	}
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for runtime")
	}

	return nil
}

func (s *policyValidator) getAllowedLifecyclesForPolicy(policy *storage.Policy) []storage.LifecycleStage {
	var lifecycleStages []storage.LifecycleStage
	if err := s.compilesForBuildTime(policy); err == nil {
		lifecycleStages = append(lifecycleStages, storage.LifecycleStage_BUILD)
	}
	if err := s.compilesForDeployTime(policy); err == nil {
		lifecycleStages = append(lifecycleStages, storage.LifecycleStage_DEPLOY)
	}
	if err := s.compilesForRunTime(policy); err == nil {
		lifecycleStages = append(lifecycleStages, storage.LifecycleStage_RUNTIME)
	}
	return lifecycleStages
}

var enforcementToLifecycle = map[storage.EnforcementAction]storage.LifecycleStage{
	storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT:                    storage.LifecycleStage_BUILD,
	storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 storage.LifecycleStage_DEPLOY,
	storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: storage.LifecycleStage_DEPLOY,
	storage.EnforcementAction_KILL_POD_ENFORCEMENT:                      storage.LifecycleStage_RUNTIME,
	storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT:             storage.LifecycleStage_RUNTIME,
}

func removeEnforcementForLifecycle(policy *storage.Policy, stage storage.LifecycleStage) {
	newActions := policy.EnforcementActions[:0]
	for _, ea := range policy.GetEnforcementActions() {
		if enforcementToLifecycle[ea] != stage {
			newActions = append(newActions, ea)
		}
	}
	policy.EnforcementActions = newActions
}

func (s *policyValidator) isAuditEventPolicy(policy *storage.Policy) bool {
	return policy.GetEventSource() == storage.EventSource_AUDIT_LOG_EVENT
}

func (s *policyValidator) validateEnforcement(policy *storage.Policy) error {
	if len(policy.GetEnforcementActions()) > 0 {
		for _, section := range policy.GetPolicySections() {
			for _, g := range section.GetPolicyGroups() {
				if len(s.nonEnforceablePolicyFields) > 0 {
					if _, ok := s.nonEnforceablePolicyFields[g.GetFieldName()]; ok {
						return errors.Errorf("enforcement of %s is not allowed", g.GetFieldName())
					}
				}
			}
		}
	}
	return nil
}
