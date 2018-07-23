package risk

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type Scorer struct {
	ConfiguredMultipliers  []multiplier
	UserDefinedMultipliers map[string]multiplier

	multLock sync.RWMutex
}

// NewScorer returns a new scorer that encompasses both static and user defined multipliers
func NewScorer(alertGetter AlertGetter, dnrIntegrationGetter DNRIntegrationGetter) *Scorer {
	return &Scorer{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Please do not re-order willy-nilly.
		ConfiguredMultipliers: []multiplier{
			newDNRAlertMultiplier(dnrIntegrationGetter),
			newViolationsMultiplier(alertGetter),
			newVulnerabilitiesMultiplier(),
			newServiceConfigMultiplier(),
			newReachabilityMultiplier(),
		},
		UserDefinedMultipliers: make(map[string]multiplier),
	}
}

// UpdateUserDefinedMultiplier upserts the passed multiplier
func (s *Scorer) UpdateUserDefinedMultiplier(mult *v1.Multiplier) {
	s.multLock.Lock()
	defer s.multLock.Unlock()
	s.UserDefinedMultipliers[mult.GetId()] = newUserDefinedMultiplier(mult)
}

// RemoveUserDefinedMultiplier removes the specific multiplier
func (s *Scorer) RemoveUserDefinedMultiplier(id string) {
	s.multLock.Lock()
	defer s.multLock.Unlock()
	delete(s.UserDefinedMultipliers, id)
}

func (s *Scorer) userDefinedScore(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
	s.multLock.RLock()
	defer s.multLock.RUnlock()

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
func (s *Scorer) sortedUserDefinedScore(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
	results, score := s.userDefinedScore(deployment)
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, score
}

func (s *Scorer) score(deployment *v1.Deployment) ([]*v1.Risk_Result, float32) {
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

// Score takes a deployment and evaluates its risk
func (s *Scorer) Score(deployment *v1.Deployment) *v1.Risk {
	riskResults, score := s.score(deployment)
	return &v1.Risk{
		Score:   score,
		Results: riskResults,
	}
}
