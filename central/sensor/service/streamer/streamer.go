package streamer

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Streamer represents an active client/server two way stream from senor to/from central.
type Streamer interface {
	Start(server central.SensorService_CommunicateServer)
	InjectMessage(msg *central.MsgToSensor) bool
	WaitUntilFinished() error
	Terminate(err error) bool
}

// NewStreamer creates a new instance of a Stream for the given data.
func NewStreamer(clusterID string, pl pipeline.Pipeline) Streamer {
	s := &streamerImpl{
		receiver:       NewReceiver(clusterID),
		enqueueDequeue: NewEnqueueDequeue(),
		pipeline:       NewPipeline(pl),
		sender:         NewSender(),
	}
	return s
}
