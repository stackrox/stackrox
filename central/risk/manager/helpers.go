package manager

import "github.com/stackrox/rox/generated/storage"

// GetDeploymentEffectiveRiskScore returns the effective risk score for a deployment.
// If user has adjusted the ranking, returns the adjusted score.
// Otherwise, returns the ML-calculated risk score.
func GetDeploymentEffectiveRiskScore(deployment *storage.Deployment) float32 {
	if deployment == nil {
		return 0.0
	}

	// If user has made an adjustment, use that
	if adj := deployment.GetUserRankingAdjustment(); adj != nil {
		return adj.GetEffectiveRiskScore()
	}

	// Otherwise, use the ML-calculated risk score
	return deployment.GetRiskScore()
}
