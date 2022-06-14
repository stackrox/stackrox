package resolvers

import (
	"context"

	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	deploymentSAC = sac.ForResource(resources.Deployment)
)

func filterDeploymentRisksOnScope(ctx context.Context, risks ...*storage.Risk) []*storage.Risk {
	filtered := risks[:0]
	for _, risk := range risks {
		scopeKeys := sac.KeyForNSScopedObj(risk.GetSubject())
		if ok, err := deploymentSAC.ReadAllowed(ctx, scopeKeys...); err != nil || !ok {
			continue
		}
		filtered = append(filtered, risk)
	}

	return filtered
}

func getAggregateRiskScore(risks ...*storage.Risk) float32 {
	score := float32(0.0)
	for _, risk := range risks {
		score += risk.GetScore()
	}
	return score
}

func scrubRiskFactors(risks ...*storage.Risk) {
	for _, risk := range risks {
		risk.Results = nil
	}
}
