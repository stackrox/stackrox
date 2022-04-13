package networkflowupdate

import (
	"context"

	protobuf "github.com/gogo/protobuf/types"
	networkBaselineManager "github.com/stackrox/stackrox/central/networkbaseline/manager"
	"github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/networkgraph"
)

type flowPersister interface {
	update(ctx context.Context, newFlows []*storage.NetworkFlow, updateTS *protobuf.Timestamp) error
}

func newFlowPersister(flowStore datastore.FlowDataStore, networkBaselines networkBaselineManager.Manager) flowPersister {
	return &flowPersisterImpl{
		flowStore:                 flowStore,
		baselines:                 networkBaselines,
		seenBaselineRelevantFlows: make(map[networkgraph.NetworkConnIndicator]struct{}),
	}
}
