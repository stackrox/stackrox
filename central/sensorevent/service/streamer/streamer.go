package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
)

// Streamer represents an active client/server two way stream from senor to/from central.
type Streamer interface {
	Start(stream Stream)
	WaitUntilFinished()
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
