package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

// Manager provides functions for working with per cluster SensorEvent streams.
// Layer of indirection allows us to inject data into the Sensor <-> Central stream.
type Manager interface {
	GetStreamer(clusterID string) Streamer
	CreateStreamer(clusterID string) (Streamer, error)
	RemoveStreamer(clusterID string) error
}

// NewManager creates a new manager on top of the given event store and pipeline.
// All created Streamer instances will use the given store for queueing, and process events with the given pipeline.
func NewManager(pl pipeline.Pipeline) Manager {
	return &managerImpl{
		pl: pl,

		clusterIDToStream: make(map[string]Streamer),
	}
}
