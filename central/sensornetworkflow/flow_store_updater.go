package sensornetworkflow

import (
	protobuf "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type flowStoreUpdater interface {
	update(newFlows []*v1.NetworkFlow, updateTS *protobuf.Timestamp) error
}

func newFlowStoreUpdater(flowStore store.FlowStore) flowStoreUpdater {
	return &flowStoreUpdaterImpl{
		flowStore: flowStore,
		isFirst:   true,
	}
}
