package networkflowupdate

import (
	"context"

	protobuf "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/generated/storage"
)

type flowStoreUpdater interface {
	update(ctx context.Context, newFlows []*storage.NetworkFlow, updateTS *protobuf.Timestamp) error
}

func newFlowStoreUpdater(flowStore datastore.FlowDataStore) flowStoreUpdater {
	return &flowStoreUpdaterImpl{
		flowStore: flowStore,
		isFirst:   true,
	}
}
