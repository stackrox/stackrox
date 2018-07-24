package risk

import (
	"bitbucket.org/stack-rox/apollo/central/risk/getters"
	"bitbucket.org/stack-rox/apollo/central/risk/multipliers"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type Scorer interface {
	Score(deployment *v1.Deployment) *v1.Risk

	UpdateUserDefinedMultiplier(mult *v1.Multiplier)
	RemoveUserDefinedMultiplier(id string)
}

// NewScorer returns a new scorer that encompasses both static and user defined multipliers
func NewScorer(alertGetter getters.AlertGetter, dnrIntegrationGetter getters.DNRIntegrationGetter) Scorer {
	return &scoreImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Please do not re-order willy-nilly.
		ConfiguredMultipliers: []multipliers.Multiplier{
			multipliers.NewDNRAlert(dnrIntegrationGetter),
			multipliers.NewViolations(alertGetter),
			multipliers.NewVulnerabilities(),
			multipliers.NewServiceConfig(),
			multipliers.NewReachability(),
		},
		UserDefinedMultipliers: make(map[string]multipliers.Multiplier),
	}
}
