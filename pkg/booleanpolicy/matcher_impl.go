package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
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

func (p *processMatcherImpl) MatchDeploymentWithProcess(cache *CacheReceptacle, deployment *storage.Deployment, images []*storage.Image, indicator *storage.ProcessIndicator, processNotInBaseline bool) (Violations, error) {
	if cache == nil || cache.augmentedObj == nil {
		processMatched, err := p.checkWhetherProcessMatches(cache, indicator, processNotInBaseline)
		if err != nil || !processMatched {
			return Violations{}, err
		}
	}

	violations, err := p.matcherImpl.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructDeploymentWithProcess(deployment, images, indicator, processNotInBaseline)
	}, indicator, nil)
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
	}, nil, event)
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

type matcherImpl struct {
	evaluators []sectionAndEvaluator
	stage      storage.LifecycleStage
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
		return augmentedobjs.ConstructImage(image)
	}, nil, nil)
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
) (*Violations, error) {
	obj, err := getOrConstructAugmentedObj(cache, constructor)
	if err != nil {
		return nil, err
	}
	v := &Violations{}
	var atLeastOneMatched bool
	var processIndicatorMatched, kubeEventMatched bool
	for _, eval := range m.evaluators {
		result, err := matchWithEvaluator(eval, obj)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}

		alertViolations, isProcessViolation, isKubeEventViolation, err :=
			violationmessages.Render(m.stage, eval.section, result, indicator, kubeEvent)
		if err != nil {
			return nil, err
		}
		if len(alertViolations) > 0 {
			atLeastOneMatched = true
		}
		if isProcessViolation {
			processIndicatorMatched = true
		} else if isKubeEventViolation {
			kubeEventMatched = true
		}

		v.AlertViolations = append(v.AlertViolations, alertViolations...)
	}
	if !atLeastOneMatched && !processIndicatorMatched && !kubeEventMatched {
		return nil, nil
	}

	if processIndicatorMatched {
		v.ProcessViolation = &storage.Alert_ProcessViolation{Processes: []*storage.ProcessIndicator{indicator}}
		printer.UpdateProcessAlertViolationMessage(v.ProcessViolation)
	} else if kubeEventMatched {
		v.AlertViolations = append(v.AlertViolations, printer.GenerateKubeEventViolationMsg(kubeEvent))
	}
	return v, nil
}

// MatchDeployment runs detection against the deployment and images.
func (m *matcherImpl) MatchDeployment(cache *CacheReceptacle, deployment *storage.Deployment, images []*storage.Image) (Violations, error) {
	violations, err := m.getViolations(cache, func() (*pathutil.AugmentedObj, error) {
		return augmentedobjs.ConstructDeployment(deployment, images)
	}, nil, nil)
	if err != nil || violations == nil {
		return Violations{}, err
	}
	return *violations, nil
}
