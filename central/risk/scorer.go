package risk

import (
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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
func NewScorer(alertGetter getters.AlertGetter) Scorer {
	return &scoreImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Please do not re-order willy-nilly.
		ConfiguredMultipliers: []multipliers.Multiplier{
			multipliers.NewViolations(alertGetter),
			multipliers.NewVulnerabilities(),
			multipliers.NewServiceConfig(),
			multipliers.NewReachability(),
		},
		UserDefinedMultipliers: make(map[string]multipliers.Multiplier),
	}
}
