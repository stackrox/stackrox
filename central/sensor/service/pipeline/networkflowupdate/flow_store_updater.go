package networkflowupdate

import (
	"context"
	"time"

	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

type flowPersister interface {
	update(ctx context.Context, newFlows []*storage.NetworkFlow, updateTS *time.Time) error
}

func newFlowPersister(flowStore datastore.FlowDataStore, networkBaselines networkBaselineManager.Manager, entityStore entityDataStore.EntityDataStore, clusterID string) flowPersister {
	return &flowPersisterImpl{
		flowStore:                 flowStore,
		baselines:                 networkBaselines,
		seenBaselineRelevantFlows: make(map[networkgraph.NetworkConnIndicator]struct{}),
		entityStore:               entityStore,
		clusterID:                 clusterID,
	}
}
