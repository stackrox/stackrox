package deployment

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/evaluator"
	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers/deployment"
	"github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating deployment risk
type Scorer interface {
	Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk
}

// NewDeploymentScorer returns a new scorer that encompasses multipliers for evaluating deployment risk
func NewDeploymentScorer(alertSearcher getters.AlertSearcher, allowlistEvaluator evaluator.Evaluator) Scorer {
	return &deploymentScorerImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Order aligns with the maximum output multiplier value, which would make sense to correlate
		// with how important a specific multiplier is.
		// DO NOT REORDER WITHOUT THOUGHT.
		ConfiguredMultipliers: []deployment.Multiplier{
			deployment.NewViolations(alertSearcher),
			//deployment.NewProcessBaselines(allowlistEvaluator),
			deployment.NewImageMultiplier(image.VulnerabilitiesHeading),
			deployment.NewServiceConfig(),
			deployment.NewReachability(),
			deployment.NewImageMultiplier(image.RiskyComponentCountHeading),
			deployment.NewImageMultiplier(image.ComponentCountHeading),
			deployment.NewImageMultiplier(image.ImageAgeHeading),
		},
	}
}

type deploymentScorerImpl struct {
	ConfiguredMultipliers []deployment.Multiplier
}

// Score takes a deployment and evaluates its risk
func (s *deploymentScorerImpl) Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	imageRiskResults := make(map[string][]*storage.Risk_Result)
	for _, risk := range images {
		for _, result := range risk.GetResults() {
			imageRiskResults[result.Name] = append(imageRiskResults[result.Name], result)
		}
	}

	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, deployment, imageRiskResults); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	if len(riskResults) == 0 {
		return nil
	}

	risk := &storage.Risk{
		Score:   overallScore,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:        deployment.GetId(),
			Type:      storage.RiskSubjectType_DEPLOYMENT,
			Namespace: deployment.GetNamespace(),
			ClusterId: deployment.GetClusterId(),
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Error(err)
		return nil
	}
	risk.Id = riskID

	return risk
}
