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

// CalculatePositionChangeScore calculates the new adjusted score when moving a deployment
// up or down in the ranking. It places the deployment midway between its current position
// and the next adjacent deployment (up or down based on moveUp parameter).
// Deployments with the same score are skipped.
// If already at boundary (top when moving up, bottom when moving down), returns current score (no-op).
// No score clamping - can result in scores > 10.0 or < 0.0.
func CalculatePositionChangeScore(currentScore float32, sortedRisks []*storage.Risk, currentIndex int, moveUp bool) float32 {
	var targetScore float32
	var found bool

	if moveUp {
		// Search upward (lower indices = higher scores in descending sort)
		for i := currentIndex - 1; i >= 0; i-- {
			candidateScore := GetEffectiveScore(sortedRisks[i])
			if candidateScore > currentScore {
				// Found a deployment with higher score (skip same scores)
				targetScore = candidateScore
				found = true
				break
			}
		}
	} else {
		// Search downward (higher indices = lower scores in descending sort)
		for i := currentIndex + 1; i < len(sortedRisks); i++ {
			candidateScore := GetEffectiveScore(sortedRisks[i])
			if candidateScore < currentScore {
				// Found a deployment with lower score (skip same scores)
				targetScore = candidateScore
				found = true
				break
			}
		}
	}

	if !found {
		// At boundary - no-op (return current score unchanged)
		return currentScore
	}

	// Return midpoint between current and target (no clamping)
	return (currentScore + targetScore) / 2.0
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
