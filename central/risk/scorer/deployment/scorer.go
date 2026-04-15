package deployment

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/evaluator"
	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers/deployment"
	"github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/central/risk/scorer/plugin/builtin"
	"github.com/stackrox/rox/central/risk/scorer/plugin/registry"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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
	scorer := &deploymentScorerImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Order aligns with the maximum output multiplier value, which would make sense to correlate
		// with how important a specific multiplier is.
		// DO NOT REORDER WITHOUT THOUGHT.
		ConfiguredMultipliers: []deployment.Multiplier{
			deployment.NewViolations(alertSearcher),
			deployment.NewProcessBaselines(allowlistEvaluator),
			deployment.NewImageMultiplier(image.VulnerabilitiesHeading),
			deployment.NewServiceConfig(),
			deployment.NewReachability(),
			deployment.NewImageMultiplier(image.RiskyComponentCountHeading),
			deployment.NewImageMultiplier(image.ComponentCountHeading),
			deployment.NewImageMultiplier(image.ImageAgeHeading),
		},
		registry: registry.Singleton(),
	}

	// Register built-in plugins and set up default configs when feature is enabled
	if features.PluginRiskScoring.Enabled() {
		builtin.RegisterPlugins(scorer.registry, alertSearcher, allowlistEvaluator)
		builtin.SetupDefaultConfigs(scorer.registry)
		log.Info("Plugin-based risk scoring enabled")
	}

	return scorer
}

type deploymentScorerImpl struct {
	ConfiguredMultipliers []deployment.Multiplier
	registry              registry.Registry
}

// Score takes a deployment and evaluates its risk
func (s *deploymentScorerImpl) Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	if features.PluginRiskScoring.Enabled() {
		return s.scoreWithPlugins(ctx, deployment, images)
	}
	return s.scoreWithMultipliers(ctx, deployment, images)
}

// scoreWithMultipliers uses the legacy hardcoded multipliers
func (s *deploymentScorerImpl) scoreWithMultipliers(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	imageRiskResults := make(map[string][]*storage.Risk_Result)
	for _, risk := range images {
		for _, result := range risk.GetResults() {
			imageRiskResults[result.GetName()] = append(imageRiskResults[result.GetName()], result)
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

	return s.buildRisk(deployment, overallScore, riskResults)
}

// scoreWithPlugins uses the plugin-based scoring system
func (s *deploymentScorerImpl) scoreWithPlugins(ctx context.Context, deployment *storage.Deployment, images []*storage.Risk) *storage.Risk {
	imageRiskResults := make(map[string][]*storage.Risk_Result)
	for _, risk := range images {
		for _, result := range risk.GetResults() {
			imageRiskResults[result.GetName()] = append(imageRiskResults[result.GetName()], result)
		}
	}

	plugins := s.registry.GetEnabledPlugins()
	if len(plugins) == 0 {
		log.Warn("No enabled risk scoring plugins found")
		return nil
	}

	var riskResults []*storage.Risk_Result
	overallScore := float32(1.0)

	for _, cp := range plugins {
		result := cp.Plugin.Score(ctx, deployment, imageRiskResults)
		if result == nil {
			continue
		}

		// Apply weight to the score
		weightedScore := result.GetScore() * cp.Config.Weight
		result.Score = weightedScore

		overallScore *= weightedScore
		riskResults = append(riskResults, result)
	}

	if len(riskResults) == 0 {
		return nil
	}

	return s.buildRisk(deployment, overallScore, riskResults)
}

// buildRisk creates the final Risk object
func (s *deploymentScorerImpl) buildRisk(deployment *storage.Deployment, overallScore float32, riskResults []*storage.Risk_Result) *storage.Risk {
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
