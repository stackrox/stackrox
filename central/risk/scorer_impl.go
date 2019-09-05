package risk

import (
	"context"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type scoreImpl struct {
	ConfiguredMultipliers []multipliers.Multiplier
}

// Score takes a deployment and evaluates its risk
func (s *scoreImpl) Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) *storage.Risk {
	riskResults, score := s.score(ctx, deployment, images)
	return &storage.Risk{
		Score:   score,
		Results: riskResults,
		Subject: &storage.RiskSubject{
			Id:        deployment.GetId(),
			Namespace: deployment.GetNamespace(),
			ClusterId: deployment.GetClusterId(),
			Type:      storage.RiskSubjectType_DEPLOYMENT,
		},
	}
}

func (s *scoreImpl) score(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Risk_Result, float32) {
	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, deployment, images); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}

	return riskResults, overallScore
}
