package manager

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
)

// GetEffectiveScore returns the effective risk score for display purposes.
// If a user ranking adjustment exists, it returns the adjusted score.
// Otherwise, it returns the original ML-calculated score.
func GetEffectiveScore(risk *storage.Risk) float32 {
	if risk == nil {
		return 0.0
	}

	if adj := risk.GetUserRankingAdjustment(); adj != nil && adj.GetLastAdjusted() != nil {
		// User has adjusted this risk - use adjusted score
		return adj.GetAdjustedScore()
	}

	// No adjustment - use original ML score
	return risk.GetScore()
}

// CalculateUpvoteScore calculates the new adjusted score when upvoting a deployment.
// It places the deployment midway between its current position and the next higher deployment.
// Deployments with the same score are skipped.
// If already at the top, returns the current score unchanged (no-op).
// No score clamping is applied - can result in scores > 10.0.
func CalculateUpvoteScore(currentScore float32, sortedRisks []*storage.Risk, currentIndex int) float32 {
	// Find next higher deployment with a different score
	var higherScore float32
	foundHigher := false

	for i := currentIndex - 1; i >= 0; i-- {
		candidateScore := GetEffectiveScore(sortedRisks[i])
		if candidateScore > currentScore {
			// Found a deployment with higher score (skip same scores)
			higherScore = candidateScore
			foundHigher = true
			break
		}
	}

	if !foundHigher {
		// Already at top - no-op (return current score unchanged)
		return currentScore
	}

	// Place midway between current and higher score
	// No clamping - allow scores to exceed 10.0
	return (currentScore + higherScore) / 2.0
}

// CalculateDownvoteScore calculates the new adjusted score when downvoting a deployment.
// It places the deployment midway between its current position and the next lower deployment.
// Deployments with the same score are skipped.
// If already at the bottom, returns the current score unchanged (no-op).
// No score clamping is applied - can result in scores < 0.0.
func CalculateDownvoteScore(currentScore float32, sortedRisks []*storage.Risk, currentIndex int) float32 {
	// Find next lower deployment with a different score
	var lowerScore float32
	foundLower := false

	for i := currentIndex + 1; i < len(sortedRisks); i++ {
		candidateScore := GetEffectiveScore(sortedRisks[i])
		if candidateScore < currentScore {
			// Found a deployment with lower score (skip same scores)
			lowerScore = candidateScore
			foundLower = true
			break
		}
	}

	if !foundLower {
		// Already at bottom - no-op (return current score unchanged)
		return currentScore
	}

	// Place midway between current and lower score
	// No clamping - allow scores to go below 0.0
	return (currentScore + lowerScore) / 2.0
}

// SortRisksByEffectiveScore sorts risks by their effective score (descending).
// This is used to determine current ranking position for upvote/downvote operations.
func SortRisksByEffectiveScore(risks []*storage.Risk) []*storage.Risk {
	sorted := make([]*storage.Risk, len(risks))
	copy(sorted, risks)

	sort.Slice(sorted, func(i, j int) bool {
		// Sort descending by effective score (highest first)
		return GetEffectiveScore(sorted[i]) > GetEffectiveScore(sorted[j])
	})

	return sorted
}

// FindDeploymentIndex finds the index of a deployment in a sorted risk list.
// Returns -1 if not found.
func FindDeploymentIndex(sortedRisks []*storage.Risk, deploymentID string) int {
	for i, risk := range sortedRisks {
		if risk.GetSubject().GetId() == deploymentID {
			return i
		}
	}
	return -1
}
