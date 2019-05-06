package risk

import (
	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	saStore "github.com/stackrox/rox/central/serviceaccount/datastore"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating risk
type Scorer interface {
	Score(deployment *storage.Deployment) *storage.Risk

	UpdateUserDefinedMultiplier(mult *storage.Multiplier)
	RemoveUserDefinedMultiplier(id string)
}

// NewScorer returns a new scorer that encompasses both static and user defined multipliers
func NewScorer(alertGetter getters.AlertGetter, indicatorGetter getters.ProcessIndicators, whitelistGetter getters.ProcessWhitelists, roles roleStore.DataStore, bindings bindingStore.DataStore, serviceAccounts saStore.DataStore) Scorer {
	scoreImpl := &scoreImpl{
		// These multipliers are intentionally ordered based on the order that we want them to be displayed in.
		// Order aligns with the maximum output multiplier value, which would make sense to correlate
		// with how important a specific multiplier is.
		// DO NOT REORDER WITHOUT THOUGHT.
		ConfiguredMultipliers: []multipliers.Multiplier{
			multipliers.NewViolations(alertGetter),
			multipliers.NewProcessWhitelists(whitelistGetter, indicatorGetter),
			multipliers.NewVulnerabilities(),
			multipliers.NewServiceConfig(),
			multipliers.NewReachability(),
			multipliers.NewRiskyComponents(),
			multipliers.NewComponentCount(),
			multipliers.NewImageAge(),
		},
		UserDefinedMultipliers: make(map[string]multipliers.Multiplier),
	}

	if features.K8sRBAC.Enabled() {
		scoreImpl.ConfiguredMultipliers = append(scoreImpl.ConfiguredMultipliers,
			multipliers.NewSAPermissionsMultiplier(roles, bindings, serviceAccounts))
	}

	return scoreImpl
}
