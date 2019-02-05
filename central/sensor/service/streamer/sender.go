package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Sender represents an active client/server two way stream from senor to/from central.
type Sender interface {
	Start(server central.SensorService_CommunicateServer, dependents ...Stoppable)
	Stop(err error) bool
	Stopped() concurrency.ReadOnlyErrorSignal

	InjectMessage(*central.MsgToSensor) bool
}

// NewSender creates a new instance of a Stream for the given data.
func NewSender() Sender {
	return &senderImpl{
		injected: make(chan *central.MsgToSensor),

		stopC:    concurrency.NewErrorSignal(),
		stoppedC: concurrency.NewErrorSignal(),
	}
}
