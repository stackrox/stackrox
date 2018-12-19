package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Receiver represents an active client/server two way stream from senor to/from central.
type Receiver interface {
	Start(stream Stream, out chan<- *central.SensorEvent)
}

// NewReceiver creates a new instance of a Stream for the given data.
func NewReceiver(clusterID string, onFinish func()) Receiver {
	return &receiverImpl{
		clusterID: clusterID,
		onFinish:  onFinish,
	}
}
