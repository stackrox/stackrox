package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

// Manager provides functions for working with per cluster SensorEvent streams.
type Manager interface {
	CreateStreamer(clusterID string) Streamer
}

// NewManager creates a new manager on top of the given event store and pipeline.
// All created Streamer instances will use the given store for queueing, and process events with the given pipeline.
func NewManager(pl pipeline.Pipeline) Manager {
	return &managerImpl{
		pl: pl,
	}
}
