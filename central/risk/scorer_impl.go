package risk

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type scoreImpl struct {
	ConfiguredMultipliers  []multipliers.Multiplier
	UserDefinedMultipliers map[string]multipliers.Multiplier

	multiplierLock sync.RWMutex
}

// Score takes a deployment and evaluates its risk
func (s *scoreImpl) Score(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) *storage.Risk {
	riskResults, score := s.score(ctx, deployment, images)
	return &storage.Risk{
		Score:   score,
		Results: riskResults,
	}
}

// UpdateUserDefinedMultiplier upserts the passed multiplier
func (s *scoreImpl) UpdateUserDefinedMultiplier(mult *storage.Multiplier) {
	s.multiplierLock.Lock()
	defer s.multiplierLock.Unlock()

	s.UserDefinedMultipliers[mult.GetId()] = multipliers.NewUserDefined(mult)
}

// RemoveUserDefinedMultiplier removes the specific multiplier
func (s *scoreImpl) RemoveUserDefinedMultiplier(id string) {
	s.multiplierLock.Lock()
	defer s.multiplierLock.Unlock()

	delete(s.UserDefinedMultipliers, id)
}

func (s *scoreImpl) userDefinedScore(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Risk_Result, float32) {
	s.multiplierLock.RLock()
	defer s.multiplierLock.RUnlock()

	score := float32(1.0)
	userDefinedRiskResults := make([]*storage.Risk_Result, 0, len(s.UserDefinedMultipliers))
	for _, mult := range s.UserDefinedMultipliers {
		if riskResult := mult.Score(ctx, deployment, images); riskResult != nil {
			score *= riskResult.GetScore()
			userDefinedRiskResults = append(userDefinedRiskResults, riskResult)
		}
	}
	return userDefinedRiskResults, score
}

// Scores from user defined multiplies are sorted in descending order of risk score.
func (s *scoreImpl) sortedUserDefinedScore(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) ([]*storage.Risk_Result, float32) {
	results, score := s.userDefinedScore(ctx, deployment, images)
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, score
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
	userDefinedResults, userDefinedScore := s.sortedUserDefinedScore(ctx, deployment, images)
	riskResults = append(riskResults, userDefinedResults...)
	overallScore *= userDefinedScore

	return riskResults, overallScore
}
