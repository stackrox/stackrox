package sensornetworkflow

import (
	protobuf "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/storage"
)

type flowStoreUpdater interface {
	update(newFlows []*storage.NetworkFlow, updateTS *protobuf.Timestamp) error
}

func newFlowStoreUpdater(flowStore store.FlowStore) flowStoreUpdater {
	return &flowStoreUpdaterImpl{
		flowStore: flowStore,
		isFirst:   true,
	}
}
