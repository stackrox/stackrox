package booleanpolicy

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
)

var (
	deploymentEvalFactory = MustCreateFactoryWrapper(augmentedobjs.DeploymentMeta)
	processEvalFactory    = MustCreateFactoryWrapper(augmentedobjs.ProcessMeta)
	imageEvalFactory      = MustCreateFactoryWrapper(augmentedobjs.ImageMeta)
	kubeEventFactory      = MustCreateFactoryWrapper(augmentedobjs.KubeEventMeta)
	networkFlowFactory    = MustCreateFactoryWrapper(augmentedobjs.NetworkFlowMeta)
)

// A CacheReceptacle is an optional argument that can be passed to the Match* functions of the Matchers below, that
// the Match* functions can use to cache values between calls. Callers MUST ensure that they only pass the same CacheReceptable
// object to subsequent calls when the other arguments being passed are the same across the calls.
// The contents of the CacheReceptacle are intentionally opaque to callers.
// Callers can pass a `nil` CacheReceptacle to disable caching.
// The zero value is ready-to-use, and denotes an empty cache.
type CacheReceptacle struct {
	augmentedObj *pathutil.AugmentedObj

	// Used only by MatchDeploymentWithProcess
	augmentedProcess *pathutil.AugmentedObj

	// Used only by MatchKubeEvent
	augmentedKubeEvent *pathutil.AugmentedObj

	// Used only by MatchDeploymentWithNetworkFlow
	augmentedNetworkFlow *pathutil.AugmentedObj
}

// EnhancedDeployment holds the deployment object plus the additional resources used for the matching.
type EnhancedDeployment struct {
	Deployment             *storage.Deployment
	Images                 []*storage.Image
	NetworkPoliciesApplied *augmentedobjs.NetworkPoliciesApplied
}

// Violations represents a list of violation sub-objects.
type Violations struct {
	ProcessViolation *storage.Alert_ProcessViolation
	AlertViolations  []*storage.Alert_Violation
}

// An ImageMatcher matches images against a policy.
type ImageMatcher interface {
	MatchImage(cache *CacheReceptacle, image *storage.Image) (Violations, error)
}

// A DeploymentMatcher matches deployments against a policy.
type DeploymentMatcher interface {
	MatchDeployment(cache *CacheReceptacle, enhancedDeployment EnhancedDeployment) (Violations, error)
}

// A DeploymentWithProcessMatcher matches deployments, and a process, against a policy.
type DeploymentWithProcessMatcher interface {
	MatchDeploymentWithProcess(cache *CacheReceptacle, enhancedDeployment EnhancedDeployment, pi *storage.ProcessIndicator, processNotInBaseline bool) (Violations, error)
}

// A KubeEventMatcher matches kubernetes event against a policy.
type KubeEventMatcher interface {
	MatchKubeEvent(cache *CacheReceptacle, kubeEvent *storage.KubernetesEvent, kubeResource interface{}) (Violations, error)
}

// An AuditLogEventMatcher matches audit log event against a policy.
type AuditLogEventMatcher interface {
	MatchAuditLogEvent(cache *CacheReceptacle, kubeEvent *storage.KubernetesEvent) (Violations, error)
}

// A DeploymentWithNetworkFlowMatcher matches deployments, and a network flow against a policy.
type DeploymentWithNetworkFlowMatcher interface {
	MatchDeploymentWithNetworkFlowInfo(cache *CacheReceptacle, enhancedDeployment EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) (Violations, error)
}

type sectionAndEvaluator struct {
	section   *storage.PolicySection
	evaluator evaluator.Evaluator
}

// BuildKubeEventMatcher builds a KubeEventMatcher.
func BuildKubeEventMatcher(p *storage.Policy, options ...ValidateOption) (KubeEventMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY, options...)
	if err != nil {
		return nil, err
	}

	kubeEventOnlyEvaluators := make([]evaluator.Evaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		if len(section.GetPolicyGroups()) == 0 {
			return nil, errors.Errorf("no groups in section %q", section.GetSectionName())
		}

		// Conjunction of process fields and events fields is not supported.
		if !ContainsDiscreteRuntimeFieldCategorySections(p) {
			return nil, errors.New("a run time policy section must not contain both process and kubernetes event constraints")
		}

		fieldQueries, err := sectionTypeToFieldQueries(section, KubeEvent)
		if err != nil {
			return nil, errors.Wrapf(err, "converting to field queries for section %q", section.GetSectionName())
		}

		// Ignore the policy sections not containing at least one kube event field.
		if len(fieldQueries) == 0 {
			continue
		}

		eval, err := kubeEventFactory.GenerateEvaluator(&query.Query{FieldQueries: fieldQueries})
		if err != nil {
			return nil, errors.Wrapf(err, "generating kube events evaluator for section %q", section.GetSectionName())
		}
		kubeEventOnlyEvaluators = append(kubeEventOnlyEvaluators, eval)
	}

	return &kubeEventMatcherImpl{
		matcherImpl: matcherImpl{
			evaluators: sectionsAndEvals,
		},
		kubeEventOnlyEvaluators: kubeEventOnlyEvaluators,
	}, nil
}

// BuildAuditLogEventMatcher builds a AuditLogEventMatcher.
func BuildAuditLogEventMatcher(p *storage.Policy, options ...ValidateOption) (AuditLogEventMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(kubeEventFactory, p, storage.LifecycleStage_RUNTIME, options...)
	if err != nil {
		return nil, err
	}
	return &auditLogEventMatcherImpl{
		matcherImpl: matcherImpl{
			evaluators: sectionsAndEvals,
		},
	}, nil
}

// BuildDeploymentWithProcessMatcher builds a DeploymentWithProcessMatcher.
func BuildDeploymentWithProcessMatcher(p *storage.Policy, options ...ValidateOption) (DeploymentWithProcessMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY, options...)
	if err != nil {
		return nil, err
	}

	processOnlyEvaluators := make([]evaluator.Evaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		if len(section.GetPolicyGroups()) == 0 {
			return nil, errors.Errorf("no groups in section %q", section.GetSectionName())
		}

		// Conjunction of process fields and events fields is not supported.
		if !ContainsDiscreteRuntimeFieldCategorySections(p) {
			return nil, errors.New("a run time policy section must not contain both process and kubernetes event constraints")
		}

		fieldQueries, err := sectionTypeToFieldQueries(section, Process)
		if err != nil {
			return nil, errors.Wrapf(err, "converting to field queries for section %q", section.GetSectionName())
		}

		// TODO: Remove AlwaysTrue evaluator once section validation is added to prohibit deploy time only fields.
		// This section has no process-related queries. This means that we cannot rule out this policy given a
		// process alone, so we must return the always true evaluator.
		// We can also discard evaluators for other sections, since they are irrelevant.
		if len(fieldQueries) == 0 {
			processOnlyEvaluators = []evaluator.Evaluator{evaluator.AlwaysTrue}
			break
		}

		eval, err := processEvalFactory.GenerateEvaluator(&query.Query{FieldQueries: fieldQueries})
		if err != nil {
			return nil, errors.Wrapf(err, "generating process evaluator for section %q", section.GetSectionName())
		}
		processOnlyEvaluators = append(processOnlyEvaluators, eval)
	}

	return &processMatcherImpl{
		matcherImpl: matcherImpl{
			evaluators: sectionsAndEvals,
		},
		processOnlyEvaluators: processOnlyEvaluators,
	}, nil
}

// BuildDeploymentWithNetworkFlowMatcher builds a DeploymentWithNetworkFlowMatcher
func BuildDeploymentWithNetworkFlowMatcher(p *storage.Policy, options ...ValidateOption) (DeploymentWithNetworkFlowMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY, options...)
	if err != nil {
		return nil, err
	}

	networkFlowOnlyEvaluators := make([]evaluator.Evaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		if len(section.GetPolicyGroups()) == 0 {
			return nil, errors.Errorf("no groups in section %q", section.GetSectionName())
		}

		// Conjunction of process fields and events fields is not supported.
		if !ContainsDiscreteRuntimeFieldCategorySections(p) {
			return nil, errors.New("a run time policy section must not contain both process and kubernetes event constraints")
		}

		fieldQueries, err := sectionTypeToFieldQueries(section, NetworkFlow)
		if err != nil {
			return nil, errors.Wrapf(err, "converting to field queries for section %q", section.GetSectionName())
		}

		eval, err := networkFlowFactory.GenerateEvaluator(&query.Query{FieldQueries: fieldQueries})
		if err != nil {
			return nil, errors.Wrapf(err, "generating network flow evaluator for section %q", section.GetSectionName())
		}
		networkFlowOnlyEvaluators = append(networkFlowOnlyEvaluators, eval)
	}

	// Although the struct implementation is the same as matcherImpl, we should still use networkFlowMatcher
	// since it implements another check func MatchDeploymentWithNetworkFlowInfo
	return &networkFlowMatcherImpl{
		matcherImpl: matcherImpl{
			evaluators: sectionsAndEvals,
		},
		networkFlowOnlyEvaluators: networkFlowOnlyEvaluators,
	}, nil
}

// BuildDeploymentMatcher builds a matcher for deployments against the given policy,
// which must be a boolean policy.
func BuildDeploymentMatcher(p *storage.Policy, options ...ValidateOption) (DeploymentMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY, options...)
	if err != nil {
		return nil, err
	}

	return &matcherImpl{
		evaluators: sectionsAndEvals,
	}, nil
}

// BuildImageMatcher builds a matcher for images against the given policy,
// which must be a boolean policy.
func BuildImageMatcher(p *storage.Policy, options ...ValidateOption) (ImageMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(imageEvalFactory, p, storage.LifecycleStage_BUILD, options...)
	if err != nil {
		return nil, err
	}
	return &matcherImpl{
		evaluators: sectionsAndEvals,
	}, nil
}

func getSectionsAndEvals(factory evaluator.Factory, p *storage.Policy, stage storage.LifecycleStage, options ...ValidateOption) ([]sectionAndEvaluator, error) {
	if err := Validate(p, options...); err != nil {
		return nil, err
	}

	if len(p.GetPolicySections()) == 0 {
		return nil, errors.New("no policy sections")
	}
	sectionsAndEvals := make([]sectionAndEvaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		sectionQ, err := sectionToQuery(section, stage)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid policy section %q", section.GetSectionName())
		}
		eval, err := factory.GenerateEvaluator(sectionQ)
		if err != nil {
			return nil, errors.Wrapf(err, "generating evaluator for policy section %q", section.GetSectionName())
		}
		sectionsAndEvals = append(sectionsAndEvals, sectionAndEvaluator{section, eval})
	}

	return sectionsAndEvals, nil
}
