package streamer

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// Service is the struct that manages the SensorEvent API
type senderImpl struct {
	onFinish func()
}

// Start sets up the channels and signal to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *senderImpl) Start(in <-chan *central.MsgToSensor, server central.SensorService_CommunicateServer) {
	go s.pipelineToSend(in, server)
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *senderImpl) pipelineToSend(in <-chan *central.MsgToSensor, server central.SensorService_CommunicateServer) {
	// When finished, close output stream and signal.
	defer s.onFinish()

	for {
		msg, ok := <-in
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		log.Warnf("sending message to sensor: %s", proto.MarshalTextString(msg))

		if err := server.Send(msg); err != nil {
			log.Error("error sending deployment enforcement", err)
		}
	}
}
