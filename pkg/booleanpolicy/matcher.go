package booleanpolicy

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

var (
	deploymentEvalFactory = evaluator.MustCreateNewFactory(augmentedobjs.DeploymentMeta)

	processEvalFactory = evaluator.MustCreateNewFactory(augmentedobjs.ProcessMeta)

	imageEvalFactory = evaluator.MustCreateNewFactory(augmentedobjs.ImageMeta)
)

// An ImageMatcher matches images against a policy.
type ImageMatcher interface {
	MatchImage(ctx context.Context, image *storage.Image) (searchbasedpolicies.Violations, error)
}

// A DeploymentMatcher matches deployments against a policy.
type DeploymentMatcher interface {
	MatchDeployment(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (searchbasedpolicies.Violations, error)
}

// A DeploymentWithProcessMatcher matches deployments, and a process, against a policy.
type DeploymentWithProcessMatcher interface {
	MatchDeploymentWithProcess(ctx context.Context, deployment *storage.Deployment, images []*storage.Image, pi *storage.ProcessIndicator, processOutsideWhitelist bool) (searchbasedpolicies.Violations, error)
}

type sectionAndEvaluator struct {
	section   *storage.PolicySection
	evaluator evaluator.Evaluator
}

// BuildDeploymentWithProcessMatcher builds a DeploymentWithProcessMatcher.
func BuildDeploymentWithProcessMatcher(p *storage.Policy) (DeploymentWithProcessMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(&deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY)
	if err != nil {
		return nil, err
	}

	processOnlyEvaluators := make([]evaluator.Evaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		if len(section.GetPolicyGroups()) == 0 {
			return nil, errors.Errorf("no groups in section %q", section.GetSectionName())
		}
		fieldQueries, err := sectionToFieldQueries(section, &runtimeFields)
		if err != nil {
			return nil, errors.Wrapf(err, "converting to field queries for section %q", section.GetSectionName())
		}
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

	return &processMatcherImpl{matcherImpl: matcherImpl{stage: storage.LifecycleStage_DEPLOY, evaluators: sectionsAndEvals}, processOnlyEvaluators: processOnlyEvaluators}, nil
}

// BuildDeploymentMatcher builds a matcher for deployments against the given policy,
// which must be a boolean policy.
func BuildDeploymentMatcher(p *storage.Policy) (DeploymentMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(&deploymentEvalFactory, p, storage.LifecycleStage_DEPLOY)
	if err != nil {
		return nil, err
	}

	return &matcherImpl{
		evaluators: sectionsAndEvals,
		stage:      storage.LifecycleStage_DEPLOY,
	}, nil
}

// BuildImageMatcher builds a matcher for images against the given policy,
// which must be a boolean policy.
func BuildImageMatcher(p *storage.Policy) (ImageMatcher, error) {
	sectionsAndEvals, err := getSectionsAndEvals(&imageEvalFactory, p, storage.LifecycleStage_BUILD)
	if err != nil {
		return nil, err
	}
	return &matcherImpl{
		evaluators: sectionsAndEvals,
		stage:      storage.LifecycleStage_BUILD,
	}, nil
}

func getSectionsAndEvals(factory *evaluator.Factory, p *storage.Policy, stage storage.LifecycleStage) ([]sectionAndEvaluator, error) {
	if err := Validate(p); err != nil {
		return nil, err
	}

	if len(p.GetPolicySections()) == 0 {
		return nil, errors.New("no sections")
	}
	sectionsAndEvals := make([]sectionAndEvaluator, 0, len(p.GetPolicySections()))
	for _, section := range p.GetPolicySections() {
		sectionQ, err := sectionToQuery(section, stage)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid section %q", section.GetSectionName())
		}
		eval, err := factory.GenerateEvaluator(sectionQ)
		if err != nil {
			return nil, errors.Wrapf(err, "generating evaluator for section %q", section.GetSectionName())
		}
		sectionsAndEvals = append(sectionsAndEvals, sectionAndEvaluator{section, eval})
	}

	return sectionsAndEvals, nil
}
