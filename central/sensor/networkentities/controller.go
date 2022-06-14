package networkentities

import (
	"context"

	"github.com/stackrox/stackrox/central/networkpolicies/graph"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// Controller handles pushing network entities to remote clusters.
type Controller interface {
	// SyncNow pushes external network entities to remote clusters.
	SyncNow(ctx context.Context) error
}

// NewController creates and returns a new controller for network graph entities.
func NewController(clusterID string,
	netEntityMgr common.NetworkEntityManager,
	graphEvaluator graph.Evaluator,
	injector common.MessageInjector,
	stopSig concurrency.ReadOnlyErrorSignal) Controller {
	return newController(clusterID, netEntityMgr, graphEvaluator, injector, stopSig)
}
