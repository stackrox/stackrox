package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
	"github.com/stackrox/rox/generated/api/v1"
)

// Streamer represents an active client/server two way stream from senor to/from central.
type Streamer interface {
	Start(stream v1.SensorEventService_RecordEventServer)
	WaitUntilFinished()

	// FOR TESTING.
	InjectEvent(event *v1.SensorEvent) bool
	InjectEnforcement(enforcement *v1.SensorEnforcement) bool
}

// NewStreamer creates a new instance of a Stream for the given data.
func NewStreamer(clusterID string, qu queue.EventQueue, pl pipeline.Pipeline) Streamer {
	s := &streamerImpl{
		clusterID: clusterID,
		qu:        qu,
		pl:        pl,
	}
	return s
}
