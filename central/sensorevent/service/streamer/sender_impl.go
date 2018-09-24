package streamer

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Service is the struct that manages the SensorEvent API
type senderImpl struct {
	onFinish func()
}

// Start sets up the channels and signal to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *senderImpl) Start(in <-chan *v1.SensorEnforcement, stream v1.SensorEventService_RecordEventServer) {
	go s.pipelineToSend(in, stream)
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *senderImpl) pipelineToSend(in <-chan *v1.SensorEnforcement, stream v1.SensorEventService_RecordEventServer) {
	// When finished, close output stream and signal.
	defer s.onFinish()

	for {
		enforcement, ok := <-in
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		if err := stream.Send(enforcement); err != nil {
			log.Error("error sending deployment enforcement", err)
		}
	}
}
