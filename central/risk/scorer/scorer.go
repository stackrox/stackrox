package scorer

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/central/processwhitelist/evaluator"
	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	saStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/risk"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating risk.
type Scorer interface {
	Score(ctx context.Context, msg proto.Message, riskIndicators ...risk.Indicator) *storage.Risk
}

// NewScorer returns a new scorer that encompasses both static and user defined multipliers
func NewScorer(alertGetter getters.AlertGetter, roles roleStore.DataStore, bindings bindingStore.DataStore, serviceAccounts saStore.DataStore, whitelistEvaluator evaluator.Evaluator) Scorer {
	scoreImpl := &scoreImpl{
		ConfiguredMultipliers: map[string]multipliers.Multiplier{
			risk.PolicyViolations.DisplayTitle:     multipliers.NewViolations(alertGetter),
			risk.SuspiciousProcesses.DisplayTitle:  multipliers.NewProcessWhitelists(whitelistEvaluator),
			risk.ImageVulnerabilities.DisplayTitle: multipliers.NewVulnerabilities(),
			risk.ServiceConfiguration.DisplayTitle: multipliers.NewServiceConfig(),
			risk.PortExposure.DisplayTitle:         multipliers.NewReachability(),
			risk.RiskyImageComponent.DisplayTitle:  multipliers.NewRiskyComponents(),
			risk.ImageComponentCount.DisplayTitle:  multipliers.NewComponentCount(),
			risk.ImageAge.DisplayTitle:             multipliers.NewImageAge(),
		},
	}

	if features.K8sRBAC.Enabled() {
		scoreImpl.ConfiguredMultipliers[risk.RBACConfiguration.DisplayTitle] = multipliers.NewSAPermissionsMultiplier(roles, bindings, serviceAccounts)
	}

	return scoreImpl
}
