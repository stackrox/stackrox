package booleanpolicy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
)

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

func (m *matcherImpl) MatchImage(_ context.Context, image *storage.Image) (searchbasedpolicies.Violations, error) {
	obj, err := augmentedobjs.ConstructImage(image)
	if err != nil {
		return searchbasedpolicies.Violations{}, err
	}
	violations, err := m.getViolations(obj, nil)
	if err != nil || violations == nil {
		return searchbasedpolicies.Violations{}, err
	}
	return *violations, nil
}

func (m *matcherImpl) getViolations(obj *pathutil.AugmentedObj, indicator *storage.ProcessIndicator) (*searchbasedpolicies.Violations, error) {
	v := &searchbasedpolicies.Violations{}
	var atLeastOneMatched bool
	var processIndicatorMatched bool
	for _, eval := range m.evaluators {
		result, err := matchWithEvaluator(eval, obj)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		alertViolations, isProcessViolation, err := violationmessages.Render(m.stage, eval.section, result, indicator)
		if err != nil {
			return nil, err
		}
		if len(alertViolations) > 0 {
			atLeastOneMatched = true
		}
		if isProcessViolation {
			processIndicatorMatched = true
		}
		v.AlertViolations = append(v.AlertViolations, alertViolations...)
	}
	if !atLeastOneMatched && !processIndicatorMatched {
		return nil, nil
	}
	if processIndicatorMatched {
		v.ProcessViolation = &storage.Alert_ProcessViolation{Processes: []*storage.ProcessIndicator{indicator}}
		builders.UpdateRuntimeAlertViolationMessage(v.ProcessViolation)
	}
	return v, nil
}

func (m *matcherImpl) MatchDeploymentWithProcess(_ context.Context, deployment *storage.Deployment, images []*storage.Image, indicator *storage.ProcessIndicator, processOutsideWhitelist bool) (searchbasedpolicies.Violations, error) {
	obj, err := augmentedobjs.ConstructDeploymentWithProcess(deployment, images, indicator, processOutsideWhitelist)
	if err != nil {
		return searchbasedpolicies.Violations{}, err
	}
	violations, err := m.getViolations(obj, indicator)
	if err != nil || violations == nil {
		return searchbasedpolicies.Violations{}, err
	}
	return *violations, nil
}

// MatchDeployment runs detection against the deployment and images.
func (m *matcherImpl) MatchDeployment(_ context.Context, deployment *storage.Deployment, images []*storage.Image) (searchbasedpolicies.Violations, error) {
	obj, err := augmentedobjs.ConstructDeployment(deployment, images)
	if err != nil {
		return searchbasedpolicies.Violations{}, err
	}
	violations, err := m.getViolations(obj, nil)
	if err != nil || violations == nil {
		return searchbasedpolicies.Violations{}, err
	}
	return *violations, nil
}
