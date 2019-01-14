package emit

import (
	"github.com/stackrox/rox/central/sensor/service/streamer"
)

// Emitter sends messages over a sensor stream to the sensor.
type Emitter interface {
	StartScrape(clusterID, scrapeID string, expectedHosts []string) error
	KillScrape(clusterID, scrapeID string) error
}

// NewEmitter returns a new instance of a Emitter.
func NewEmitter(sensorStreamManager streamer.Manager) Emitter {
	return &emitterImpl{
		sensorStreamManager: sensorStreamManager,
	}
}
