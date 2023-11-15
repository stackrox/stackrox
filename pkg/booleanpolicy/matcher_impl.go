package booleanpolicy

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/networkbaseline"
)

var (
	log = logging.LoggerForModule()
)

type processMatcherImpl struct {
	processOnlyEvaluators []evaluator.Evaluator
	matcherImpl
}

func (p *processMatcherImpl) checkWhetherProcessMatches(cache *CacheReceptacle, indicator *storage.ProcessIndicator, processNotInBaseline bool) (bool, error) {
	var augmentedProcess *pathutil.AugmentedObj
	if cache != nil && cache.augmentedProcess != nil {
		augmentedProcess = cache.augmentedProcess
	} else {
		var err error
		augmentedProcess, err = augmentedobjs.ConstructProcess(indicator, processNotInBaseline)
		if err != nil {
			return false, err
		}
		if cache != nil {
			cache.augmentedProcess = augmentedProcess
		}
	}
	for _, eval := range p.processOnlyEvaluators {
		_, matched := eval.Evaluate(augmentedProcess.Value())
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (p *processMatcherImpl) MatchDeploymentWithProcess(cache *CacheReceptacle, enhancedDeployment EnhancedDeployment, indicator *storage.ProcessIndicator, processNotInBaseline bool) (Violations, error) {
	if cache == nil || cache.augmentedObj == nil {
		processMatched, err := p.checkWhetherProcessMatches(cache, indicator, processNotInBaseline)
		if err != nil || !processMatched {
			return Violations{}, err
		}
	}

	violations, err := p.matcherImpl.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructDeploymentWithProcess(enhancedDeployment.Deployment, enhancedDeployment.Images, enhancedDeployment.NetworkPoliciesApplied, indicator, processNotInBaseline)
	}, indicator, nil, nil, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}

type kubeEventMatcherImpl struct {
	kubeEventOnlyEvaluators []evaluator.Evaluator
	matcherImpl
}

func (m *kubeEventMatcherImpl) MatchKubeEvent(cache *CacheReceptacle, event *storage.KubernetesEvent, kubeResource interface{}) (Violations, error) {
	if cache == nil || cache.augmentedObj == nil {
		if matched, err := m.checkWhetherKubeEventMatches(cache, event); err != nil || !matched {
			return Violations{}, err
		}
	}

	violations, err := m.matcherImpl.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructKubeResourceWithEvent(kubeResource, event)
	}, nil, event, nil, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}

type auditLogEventMatcherImpl struct {
	matcherImpl
}

func (m *auditLogEventMatcherImpl) MatchAuditLogEvent(cache *CacheReceptacle, event *storage.KubernetesEvent) (Violations, error) {
	violations, err := m.matcherImpl.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructAuditEvent(event, event.ImpersonatedUser != nil)
	}, nil, event, nil, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}

func (m *kubeEventMatcherImpl) checkWhetherKubeEventMatches(cache *CacheReceptacle, event *storage.KubernetesEvent) (bool, error) {
	var augmentedEvent *pathutil.AugmentedObj
	if cache != nil && cache.augmentedKubeEvent != nil {
		augmentedEvent = cache.augmentedKubeEvent
	} else {
		augmentedEvent = augmentedobjs.ConstructKubeEvent(event)
		if cache != nil {
			cache.augmentedKubeEvent = augmentedEvent
		}
	}

	for _, eval := range m.kubeEventOnlyEvaluators {
		if _, matched := eval.Evaluate(augmentedEvent.Value()); matched {
			return true, nil
		}
	}
	return false, nil
}

type networkFlowMatcherImpl struct {
	networkFlowOnlyEvaluators []evaluator.Evaluator
	matcherImpl
}

func (m *networkFlowMatcherImpl) checkFlowEntitySupportsPolicy(t storage.NetworkEntityInfo_Type) bool {
	// For now, we only support running policy checks on flows which we also support in network baselines
	_, ok := networkbaseline.ValidBaselinePeerEntityTypes[t]
	return ok
}

func (m *networkFlowMatcherImpl) checkWhetherFlowMatches(
	cache *CacheReceptacle,
	flow *augmentedobjs.NetworkFlowDetails,
) (bool, error) {
	// First make sure both src and dst entities support policy checking
	if !m.checkFlowEntitySupportsPolicy(flow.SrcEntityType) ||
		!m.checkFlowEntitySupportsPolicy(flow.DstEntityType) {
		return false, nil
	}

	var augmentedNetworkFlow *pathutil.AugmentedObj
	if cache != nil && cache.augmentedNetworkFlow != nil {
		augmentedNetworkFlow = cache.augmentedNetworkFlow
	} else {
		var err error
		augmentedNetworkFlow, err = augmentedobjs.ConstructNetworkFlow(flow)
		if err != nil {
			return false, err
		}
		if cache != nil {
			cache.augmentedNetworkFlow = augmentedNetworkFlow
		}
	}
	for _, eval := range m.networkFlowOnlyEvaluators {
		_, matched := eval.Evaluate(augmentedNetworkFlow.Value())
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *networkFlowMatcherImpl) MatchDeploymentWithNetworkFlowInfo(
	cache *CacheReceptacle,
	enhancedDeployment EnhancedDeployment,
	flow *augmentedobjs.NetworkFlowDetails,
) (Violations, error) {
	if cache == nil || cache.augmentedObj == nil {
		processMatched, err := m.checkWhetherFlowMatches(cache, flow)
		if err != nil || !processMatched {
			return Violations{}, err
		}
	}

	violations, err := m.matcherImpl.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructDeploymentWithNetworkFlowInfo(enhancedDeployment.Deployment, enhancedDeployment.Images, enhancedDeployment.NetworkPoliciesApplied, flow)
	}, nil, nil, flow, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}

type matcherImpl struct {
	evaluators []sectionAndEvaluator
}

func matchWithEvaluator(sectionAndEval sectionAndEvaluator, obj *pathutil.AugmentedObj) (*evaluator.Result, error) {
	finalResult, matched := sectionAndEval.evaluator.Evaluate(obj.Value())
	if !matched {
		return nil, nil
	}
	return finalResult, nil
}

func (m *matcherImpl) MatchImage(cache *CacheReceptacle, image *storage.Image) (Violations, error) {
	violations, err := m.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructImage(image, image.GetName().GetFullName())
	}, nil, nil, nil, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}

// getOrConstructAugmentedObj either retrieves the augmented obj from the cache, or constructs it and adds to the cache.
// If the cache is `nil`, then the cache is ignored.
func getOrConstructAugmentedObj(cache *CacheReceptacle, constructor func() (*pathutil.AugmentedObj, error)) (*pathutil.AugmentedObj, error) {
	if cache == nil {
		return constructor()
	}
	if cache.augmentedObj != nil {
		return cache.augmentedObj, nil
	}
	obj, err := constructor()
	if err != nil {
		return nil, err
	}
	cache.augmentedObj = obj
	return obj, nil
}

func (m *matcherImpl) getViolations(
	cache *CacheReceptacle,
	constructor func() (*pathutil.AugmentedObj, error),
	indicator *storage.ProcessIndicator,
	kubeEvent *storage.KubernetesEvent,
	networkFlow *augmentedobjs.NetworkFlowDetails,
	networkPolicy *augmentedobjs.NetworkPoliciesApplied,
) (*Violations, error) {
	obj, err := getOrConstructAugmentedObj(cache, constructor)
	if err != nil {
		return nil, err
	}
	v := &Violations{}
	var atLeastOneMatched bool
	var processIndicatorMatched, kubeOrAuditEventMatched, networkFlowMatched, networkPolicyMatched bool
	for _, eval := range m.evaluators {
		result, err := matchWithEvaluator(eval, obj)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}

		alertViolations, isProcessViolation, isKubeOrAuditEventViolation, isNetworkFlowViolation, isNetworkPolicyViolation, err :=
			violationmessages.Render(eval.section, result, indicator, kubeEvent, networkFlow, networkPolicy)
		if err != nil {
			return nil, err
		}
		if len(alertViolations) > 0 {
			atLeastOneMatched = true
		}
		if isProcessViolation {
			processIndicatorMatched = true
		} else if isKubeOrAuditEventViolation {
			kubeOrAuditEventMatched = true
		} else if isNetworkFlowViolation {
			networkFlowMatched = true
		} else if isNetworkPolicyViolation {
			networkPolicyMatched = true
		}

		v.AlertViolations = append(v.AlertViolations, alertViolations...)
	}
	if !atLeastOneMatched && !processIndicatorMatched && !kubeOrAuditEventMatched && !networkFlowMatched && !networkPolicyMatched {
		return nil, nil
	}

	if processIndicatorMatched {
		v.ProcessViolation = &storage.Alert_ProcessViolation{Processes: []*storage.ProcessIndicator{indicator}}
		printer.UpdateProcessAlertViolationMessage(v.ProcessViolation)
	} else if kubeOrAuditEventMatched {
		v.AlertViolations = append(v.AlertViolations, printer.GenerateKubeEventViolationMsg(kubeEvent))
	} else if networkFlowMatched {
		networkFlowViolationMsg, err := printer.GenerateNetworkFlowViolation(networkFlow)
		if err != nil {
			return nil, errors.Wrap(err, "generating network flow violation message")
		}
		v.AlertViolations = append(v.AlertViolations, networkFlowViolationMsg)
	} else if networkPolicyMatched {
		v.AlertViolations = printer.EnhanceNetworkPolicyViolations(v.AlertViolations, networkPolicy)
	}
	return v, nil
}

// MatchDeployment runs detection against the deployment and images.
func (m *matcherImpl) MatchDeployment(cache *CacheReceptacle, enhancedDeployment EnhancedDeployment) (Violations, error) {
	violations, err := m.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructDeployment(enhancedDeployment.Deployment, enhancedDeployment.Images, enhancedDeployment.NetworkPoliciesApplied)
	}, nil, nil, nil, enhancedDeployment.NetworkPoliciesApplied)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}
