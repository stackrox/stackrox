package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/policies"
)

func newPolicyValidator(notifierStorage notifierDataStore.DataStore, clusterStorage clusterDataStore.DataStore, deploymentMatcherBuilder, imageMatcherBuilder matcher.Builder) *policyValidator {
	return &policyValidator{
		notifierStorage:          notifierStorage,
		clusterStorage:           clusterStorage,
		deploymentMatcherBuilder: deploymentMatcherBuilder,
		imageMatcherBuilder:      imageMatcherBuilder,
		nameValidator:            regexp.MustCompile(`^[^\n\r\$]{5,64}$`),
		descriptionValidator:     regexp.MustCompile(`^[^\$]{1,256}$`),
	}
}

// policyValidator validates the incoming policy.
type policyValidator struct {
	notifierStorage          notifierDataStore.DataStore
	clusterStorage           clusterDataStore.DataStore
	deploymentMatcherBuilder matcher.Builder
	imageMatcherBuilder      matcher.Builder

	nameValidator        *regexp.Regexp
	descriptionValidator *regexp.Regexp
}

func (s *policyValidator) validate(ctx context.Context, policy *storage.Policy) error {
	s.removeEnforcementsForMissingLifecycles(policy)

	errorList := errorhelpers.NewErrorList("policy invalid")
	errorList.AddError(s.validateName(policy))
	errorList.AddError(s.validateDescription(policy))
	errorList.AddError(s.validateCompilableForLifecycle(policy))
	errorList.AddError(s.validateSeverity(policy))
	errorList.AddError(s.validateCategories(policy))
	errorList.AddError(s.validateScopes(policy))
	errorList.AddError(s.validateWhitelists(policy))
	errorList.AddError(s.validateCapabilities(policy))
	errorList.AddError(s.validateNotifiers(ctx, policy))
	return errorList.ToError()
}

func (s *policyValidator) validateName(policy *storage.Policy) error {
	if policy.GetName() == "" || !s.nameValidator.MatchString(policy.GetName()) {
		return errors.New("policy must have a name, at least 5 chars long, and contain no punctuation or special characters")
	}
	return nil
}

func (s *policyValidator) validateDescription(policy *storage.Policy) error {
	if policy.GetDescription() != "" && !s.descriptionValidator.MatchString(policy.GetDescription()) {
		return errors.New("description, when present, should be of sentence form, and not contain more than 200 characters")
	}
	return nil
}

func (s *policyValidator) validateCompilableForLifecycle(policy *storage.Policy) error {
	if len(policy.GetLifecycleStages()) == 0 {
		return fmt.Errorf("a policy must apply to at least one lifecycle stage")
	}

	errorList := errorhelpers.NewErrorList("error validating lifecycle stage")
	if policies.AppliesAtBuildTime(policy) {
		errorList.AddError(s.compilesForBuildTime(policy))
	}
	if policies.AppliesAtDeployTime(policy) {
		errorList.AddError(s.compilesForDeployTime(policy))
	}
	if policies.AppliesAtRunTime(policy) {
		errorList.AddError(s.compilesForRunTime(policy))
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

func (s *policyValidator) validateSeverity(policy *storage.Policy) error {
	if policy.GetSeverity() == storage.Severity_UNSET_SEVERITY {
		return errors.New("a policy must have a severity")
	}
	return nil
}

func (s *policyValidator) validateCapabilities(policy *storage.Policy) error {
	set := mapset.NewSet()
	for _, s := range policy.GetFields().GetAddCapabilities() {
		set.Add(s)
	}
	var duplicates []string
	for _, s := range policy.GetFields().GetDropCapabilities() {
		if set.Contains(s) {
			duplicates = append(duplicates, s)
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

func (s *policyValidator) validateWhitelists(policy *storage.Policy) error {
	for _, whitelist := range policy.GetWhitelists() {
		if err := s.validateWhitelist(policy, whitelist); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateWhitelist(policy *storage.Policy, whitelist *storage.Whitelist) error {
	// TODO(cgorman) once we have real whitelist support in UI, add validation for whitelist name
	if whitelist.GetDeployment() == nil && whitelist.GetImage() == nil {
		return errors.New("all whitelists must have some criteria to match on")
	}
	if whitelist.GetDeployment() != nil {
		if !policies.AppliesAtDeployTime(policy) && !policies.AppliesAtRunTime(policy) {
			return errors.New("whitelisting a deployment is only valid during the DEPLOY and RUNTIME lifecycles")
		}
		if err := s.validateDeploymentWhitelist(whitelist); err != nil {
			return err
		}
	}
	if whitelist.GetImage() != nil {
		if !policies.AppliesAtBuildTime(policy) {
			return errors.New("whitelisting an image is only valid during the BUILD lifecycle")
		}
		if whitelist.GetImage().GetName() == "" {
			return fmt.Errorf("image whitelist must have nonempty name")
		}
	}
	return nil
}

func (s *policyValidator) validateDeploymentWhitelist(whitelist *storage.Whitelist) error {
	deployment := whitelist.GetDeployment()
	if deployment.GetScope() == nil && deployment.GetName() == "" {
		return errors.New("at least one field of deployment whitelist must be defined")
	}
	if deployment.GetScope() != nil {
		if err := s.validateScope(deployment.GetScope()); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateScope(scope *storage.Scope) error {
	if scope.GetCluster() == "" && scope.GetNamespace() == "" && scope.GetLabel() == nil {
		return errors.New("scope must have at least one field populated")
	}
	return nil
}

func (s *policyValidator) compilesForBuildTime(policy *storage.Policy) error {
	m, err := s.imageMatcherBuilder.ForPolicy(policy)
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for build time")
	}
	if m == nil {
		return errors.New("build time policy contains no image constraints")
	}
	return nil
}

func (s *policyValidator) compilesForDeployTime(policy *storage.Policy) error {
	m, err := s.deploymentMatcherBuilder.ForPolicy(policy)
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for deploy time")
	}
	if m == nil {
		return fmt.Errorf("deploy time policy contains no constraints")
	}
	if policy.GetFields().GetProcessPolicy() != nil {
		return errors.New("deploy time policy cannot contain runtime fields")
	}
	return nil
}

func (s *policyValidator) compilesForRunTime(policy *storage.Policy) error {
	m, err := s.deploymentMatcherBuilder.ForPolicy(policy)
	if err != nil {
		return errors.Wrap(err, "policy configuration is invalid for run time")
	}
	if m == nil {
		return errors.New("run time policy contains no constraints")
	}
	if policy.GetFields().GetProcessPolicy() == nil && !policy.GetFields().GetWhitelistEnabled() {
		return errors.New("run time policy must contain runtime specific constraints")
	}
	return nil
}

var enforcementToLifecycle = map[storage.EnforcementAction]storage.LifecycleStage{
	storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT:                    storage.LifecycleStage_BUILD,
	storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 storage.LifecycleStage_DEPLOY,
	storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: storage.LifecycleStage_DEPLOY,
	storage.EnforcementAction_KILL_POD_ENFORCEMENT:                      storage.LifecycleStage_RUNTIME,
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
