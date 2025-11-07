package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func createRisk(id string, score float32) *storage.Risk {
	return &storage.Risk{
		Subject: &storage.RiskSubject{
			Id:   id,
			Type: storage.RiskSubjectType_DEPLOYMENT,
		},
		Score: score,
	}
}

func createRiskWithAdjustment(id string, originalScore float32, adjustedScore float32) *storage.Risk {
	risk := createRisk(id, originalScore)
	risk.UserRankingAdjustment = &storage.UserRankingAdjustment{
		AdjustedScore: adjustedScore,
		LastAdjusted:  protocompat.TimestampNow(),
	}
	return risk
}

func TestGetEffectiveScore(t *testing.T) {
	t.Run("returns original score when no adjustment", func(t *testing.T) {
		risk := createRisk("deploy-1", 5.0)
		assert.Equal(t, float32(5.0), GetEffectiveScore(risk))
	})

	t.Run("returns adjusted score when adjustment exists", func(t *testing.T) {
		risk := createRiskWithAdjustment("deploy-1", 5.0, 7.5)
		assert.Equal(t, float32(7.5), GetEffectiveScore(risk))
	})

	t.Run("returns 0 for nil risk", func(t *testing.T) {
		assert.Equal(t, float32(0.0), GetEffectiveScore(nil))
	})
}

func TestCalculatePositionChangeScore(t *testing.T) {
	t.Run("moving up places deployment midway between current and higher score", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 4.0),
		}

		// Move deploy-2 up (score 6.0) - should place it between 6.0 and 8.0
		newScore := CalculatePositionChangeScore(6.0, risks, 1, true)
		assert.Equal(t, float32(7.0), newScore) // (6.0 + 8.0) / 2 = 7.0
	})

	t.Run("moving up skips deployments with same score", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 6.0), // Same as current
			createRisk("deploy-4", 4.0),
		}

		// Move deploy-3 up (score 6.0) - should skip deploy-2 (also 6.0) and use deploy-1 (8.0)
		newScore := CalculatePositionChangeScore(6.0, risks, 2, true)
		assert.Equal(t, float32(7.0), newScore) // (6.0 + 8.0) / 2 = 7.0
	})

	t.Run("moving up returns current score when already at top", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 4.0),
		}

		// Move deploy-1 up (already at top) - should be no-op
		newScore := CalculatePositionChangeScore(8.0, risks, 0, true)
		assert.Equal(t, float32(8.0), newScore)
	})

	t.Run("moving up allows scores to exceed 10.0", func(t *testing.T) {
		risks := []*storage.Risk{
			createRiskWithAdjustment("deploy-1", 9.0, 11.0), // Already adjusted above 10
			createRisk("deploy-2", 9.0),
		}

		// Move deploy-2 up - should place it above 10.0
		newScore := CalculatePositionChangeScore(9.0, risks, 1, true)
		assert.Equal(t, float32(10.0), newScore) // (9.0 + 11.0) / 2 = 10.0
	})

	t.Run("moving down places deployment midway between current and lower score", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 4.0),
		}

		// Move deploy-2 down (score 6.0) - should place it between 6.0 and 4.0
		newScore := CalculatePositionChangeScore(6.0, risks, 1, false)
		assert.Equal(t, float32(5.0), newScore) // (6.0 + 4.0) / 2 = 5.0
	})

	t.Run("moving down skips deployments with same score", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 6.0), // Same as current
			createRisk("deploy-4", 4.0),
		}

		// Move deploy-2 down (score 6.0) - should skip deploy-3 (also 6.0) and use deploy-4 (4.0)
		newScore := CalculatePositionChangeScore(6.0, risks, 1, false)
		assert.Equal(t, float32(5.0), newScore) // (6.0 + 4.0) / 2 = 5.0
	})

	t.Run("moving down returns current score when already at bottom", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 8.0),
			createRisk("deploy-2", 6.0),
			createRisk("deploy-3", 4.0),
		}

		// Move deploy-3 down (already at bottom) - should be no-op
		newScore := CalculatePositionChangeScore(4.0, risks, 2, false)
		assert.Equal(t, float32(4.0), newScore)
	})

	t.Run("moving down allows scores to go below 0.0", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 2.0),
			createRiskWithAdjustment("deploy-2", 1.0, -1.0), // Already adjusted below 0
		}

		// Move deploy-1 down - should place it below 0.0
		newScore := CalculatePositionChangeScore(2.0, risks, 0, false)
		assert.Equal(t, float32(0.5), newScore) // (2.0 + (-1.0)) / 2 = 0.5
	})
}

func TestSortRisksByEffectiveScore(t *testing.T) {
	t.Run("sorts by effective score descending", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 4.0),
			createRiskWithAdjustment("deploy-2", 3.0, 9.0), // Adjusted to 9.0
			createRisk("deploy-3", 6.0),
		}

		sorted := SortRisksByEffectiveScore(risks)

		assert.Len(t, sorted, 3)
		assert.Equal(t, "deploy-2", sorted[0].GetSubject().GetId()) // 9.0 (adjusted)
		assert.Equal(t, "deploy-3", sorted[1].GetSubject().GetId()) // 6.0
		assert.Equal(t, "deploy-1", sorted[2].GetSubject().GetId()) // 4.0
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		risks := []*storage.Risk{
			createRisk("deploy-1", 4.0),
			createRisk("deploy-2", 6.0),
		}

		sorted := SortRisksByEffectiveScore(risks)

		// Original should be unchanged
		assert.Equal(t, "deploy-1", risks[0].GetSubject().GetId())
		assert.Equal(t, "deploy-2", risks[1].GetSubject().GetId())

		// Sorted should be different order
		assert.Equal(t, "deploy-2", sorted[0].GetSubject().GetId())
		assert.Equal(t, "deploy-1", sorted[1].GetSubject().GetId())
	})
}

func TestFindDeploymentIndex(t *testing.T) {
	risks := []*storage.Risk{
		createRisk("deploy-1", 8.0),
		createRisk("deploy-2", 6.0),
		createRisk("deploy-3", 4.0),
	}

	t.Run("finds deployment at start", func(t *testing.T) {
		idx := FindDeploymentIndex(risks, "deploy-1")
		assert.Equal(t, 0, idx)
	})

	t.Run("finds deployment in middle", func(t *testing.T) {
		idx := FindDeploymentIndex(risks, "deploy-2")
		assert.Equal(t, 1, idx)
	})

	t.Run("finds deployment at end", func(t *testing.T) {
		idx := FindDeploymentIndex(risks, "deploy-3")
		assert.Equal(t, 2, idx)
	})

	t.Run("returns -1 for not found", func(t *testing.T) {
		idx := FindDeploymentIndex(risks, "deploy-999")
		assert.Equal(t, -1, idx)
	})
}
