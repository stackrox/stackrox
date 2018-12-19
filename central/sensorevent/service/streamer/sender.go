package streamer

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Sender represents an active client/server two way stream from senor to/from central.
type Sender interface {
	Start(in <-chan *v1.SensorEnforcement, stream Stream)
}

// NewSender creates a new instance of a Stream for the given data.
func NewSender(onFinish func()) Sender {
	return &senderImpl{
		onFinish: onFinish,
	}
}
