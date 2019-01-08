package streamer

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

// Manager provides functions for working with per cluster SensorEvent streams.
type Manager interface {
	CreateStreamer(clusterID string) (Streamer, error)
	GetStreamer(clusterID string) Streamer
	RemoveStreamer(clusterID string, streamer Streamer) error
}

// NewManager creates a new manager on top of the given pipeline factory.
func NewManager(pf pipeline.Factory) Manager {
	return &managerImpl{
		streamers: make(map[string]Streamer),
		pf:        pf,
	}
}
