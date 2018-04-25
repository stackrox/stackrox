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
	ConfiguredMultipliers  []Multiplier
	UserDefinedMultipliers map[string]Multiplier

	multLock sync.Mutex
}

// NewScorer returns a new scorer that encompasses both static and user defined multipliers
func NewScorer(getter AlertGetter) *Scorer {
	return &Scorer{
		ConfiguredMultipliers: []Multiplier{
			NewServiceConfigMultiplier(),
			NewVulnerabilitiesMultiplier(),
			NewViolationsMultiplier(getter),
		},
		UserDefinedMultipliers: make(map[string]Multiplier),
	}
}

// UpdateUserDefinedMultiplier upserts the passed multiplier
func (s *Scorer) UpdateUserDefinedMultiplier(mult *v1.Multiplier) {
	s.multLock.Lock()
	defer s.multLock.Unlock()
	s.UserDefinedMultipliers[mult.GetId()] = NewUserDefinedMultiplier(mult)
}

// RemoveUserDefinedMultiplier removes the specific multiplier
func (s *Scorer) RemoveUserDefinedMultiplier(id string) {
	s.multLock.Lock()
	defer s.multLock.Unlock()
	delete(s.UserDefinedMultipliers, id)
}

// Score takes a deployment and evaluates its risk
func (s *Scorer) Score(deployment *v1.Deployment) *v1.Risk {
	s.multLock.Lock()
	defer s.multLock.Unlock()
	riskResults := make([]*v1.Risk_Result, 0, len(s.ConfiguredMultipliers)+len(s.UserDefinedMultipliers))
	overallScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(deployment); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	for _, mult := range s.UserDefinedMultipliers {
		if riskResult := mult.Score(deployment); riskResult != nil {
			overallScore *= riskResult.GetScore()
			riskResults = append(riskResults, riskResult)
		}
	}
	sort.SliceStable(riskResults, func(i, j int) bool { return riskResults[i].Score > riskResults[j].Score })
	return &v1.Risk{
		Score:   overallScore,
		Results: riskResults,
	}
}
