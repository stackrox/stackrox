package sensornetworkflow

import "github.com/stackrox/rox/central/networkflow/store"

// Handler takes care of receiving flows from a Stream, and making the necessary changes to the flow store.
type Handler interface {
	Run() error
}

// NewHandler creates and returns a new networkflow handler.
func NewHandler(clusterID string, flowStore store.FlowStore, stream Stream) Handler {
	return &handler{
		clusterID: clusterID,
		updater:   newFlowStoreUpdater(flowStore),
		stream:    stream,
	}
}
