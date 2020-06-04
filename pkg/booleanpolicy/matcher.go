package booleanpolicy

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

var (
	deploymentEvalFactory = evaluator.MustCreateNewFactory(augmentedobjs.DeploymentMeta)

	imageEvalFactory = evaluator.MustCreateNewFactory(augmentedobjs.ImageMeta)
)

// An ImageMatcher matches images against a policy.
type ImageMatcher interface {
	MatchImage(ctx context.Context, image *storage.Image) (searchbasedpolicies.Violations, error)
}

// A DeploymentMatcher matches deployments against a policy.
type DeploymentMatcher interface {
	MatchDeployment(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (searchbasedpolicies.Violations, error)
	MatchDeploymentWithProcess(ctx context.Context, deployment *storage.Deployment, images []*storage.Image, pi *storage.ProcessIndicator, processOutsideWhitelist bool) (searchbasedpolicies.Violations, error)
}

type sectionAndEvaluator struct {
	section   *storage.PolicySection
	evaluator evaluator.Evaluator
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
