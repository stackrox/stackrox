package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Sender represents an active client/server two way stream from senor to/from central.
type Sender interface {
	Start(in <-chan *central.MsgToSensor, server central.SensorService_CommunicateServer)
}

// NewSender creates a new instance of a Stream for the given data.
func NewSender(onFinish func()) Sender {
	return &senderImpl{
		onFinish: onFinish,
	}
}
