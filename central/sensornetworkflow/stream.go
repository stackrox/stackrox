package sensornetworkflow

import "github.com/stackrox/rox/generated/internalapi/central"

// Stream is an abstraction for a stream over which to receive network flow updates.
type Stream interface {
	Recv() (*central.NetworkFlowUpdate, error)
}
