package risk

import (
	"sort"
	"sync"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/api/v1"
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type scoreImpl struct {
	ConfiguredMultipliers  []multipliers.Multiplier
	UserDefinedMultipliers map[string]multipliers.Multiplier

	multiplierLock sync.RWMutex
}

// Score takes a deployment and evaluates its risk
func (s *scoreImpl) Score(deployment *v1.Deployment) *v1.Risk {
	riskResults, score := s.score(deployment)
	return &v1.Risk{
		Score:   score,
		Results: riskResults,
	}
}

// UpdateUserDefinedMultiplier upserts the passed multiplier
func (s *scoreImpl) UpdateUserDefinedMultiplier(mult *v1.Multiplier) {
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

func (s *scoreImpl) userDefinedScore(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
	s.multiplierLock.RLock()
	defer s.multiplierLock.RUnlock()

	score := float32(1.0)
	userDefinedRiskResults := make([]*v1.Risk_Result, 0, len(s.UserDefinedMultipliers))
	for _, mult := range s.UserDefinedMultipliers {
		if riskResult := mult.Score(deployment); riskResult != nil {
			score *= riskResult.GetScore()
			userDefinedRiskResults = append(userDefinedRiskResults, riskResult)
		}
	}
	return userDefinedRiskResults, score
}

// Scores from user defined multiplies are sorted in descending order of risk score.
func (s *scoreImpl) sortedUserDefinedScore(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
	results, score := s.userDefinedScore(deployment)
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, score
}

func (s *scoreImpl) score(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
	riskResults := make([]*v1.Risk_Result, 0, len(s.ConfiguredMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(deployment); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	userDefinedResults, userDefinedScore := s.sortedUserDefinedScore(deployment)
	riskResults = append(riskResults, userDefinedResults...)
	overallScore *= userDefinedScore

	return riskResults, overallScore
}
