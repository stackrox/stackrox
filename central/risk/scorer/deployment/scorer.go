package deployment

import (
	"context"

	"github.com/stackrox/stackrox/central/processbaseline/evaluator"
	roleStore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/central/risk/getters"
	"github.com/stackrox/stackrox/central/risk/multipliers/deployment"
	"github.com/stackrox/stackrox/central/risk/multipliers/image"
	saStore "github.com/stackrox/stackrox/central/serviceaccount/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating deployment risk
type Scorer interface {
	Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk
}

// NewDeploymentScorer returns a new scorer that encompasses multipliers for evaluating deployment risk
func NewDeploymentScorer(alertGetter getters.AlertGetter, roles roleStore.DataStore, bindings bindingStore.DataStore, serviceAccounts saStore.DataStore, allowlistEvaluator evaluator.Evaluator) Scorer {
	scoreImpl := &deploymentScorerImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Order aligns with the maximum output multiplier value, which would make sense to correlate
		// with how important a specific multiplier is.
		// DO NOT REORDER WITHOUT THOUGHT.
		ConfiguredMultipliers: []deployment.Multiplier{
			deployment.NewViolations(alertGetter),
			deployment.NewProcessBaselines(allowlistEvaluator),
			deployment.NewImageMultiplier(image.VulnerabilitiesHeading),
			deployment.NewServiceConfig(),
			deployment.NewReachability(),
			deployment.NewImageMultiplier(image.RiskyComponentCountHeading),
			deployment.NewImageMultiplier(image.ComponentCountHeading),
			deployment.NewImageMultiplier(image.ImageAgeHeading),
		},
	}
	if env.IncludeRBACInRisk.BooleanSetting() {
		scoreImpl.ConfiguredMultipliers = append(scoreImpl.ConfiguredMultipliers,
			deployment.NewSAPermissionsMultiplier(roles, bindings, serviceAccounts),
		)
	}

	return scoreImpl
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
