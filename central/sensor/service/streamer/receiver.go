package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Receiver represents an active client/server two way stream from senor to/from central.
type Receiver interface {
	Start(server central.SensorService_CommunicateServer, dependents ...Stoppable)
	Stopped() concurrency.ReadOnlyErrorSignal

	Output() <-chan *central.MsgFromSensor
}

// NewReceiver creates a new instance of a Stream for the given data.
func NewReceiver(clusterID string) Receiver {
	return &receiverImpl{
		clusterID: clusterID,

		output: make(chan *central.MsgFromSensor),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
