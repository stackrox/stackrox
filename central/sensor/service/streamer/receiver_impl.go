package streamer

import (
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
)

type receiverImpl struct {
	clusterID string
	onFinish  func()
}

// Start starts receiving from the grpc stream and pushing recieved events to the out channel.
func (s *receiverImpl) Start(server central.SensorService_CommunicateServer, out chan<- *central.MsgFromSensor) {
	go s.receiveToChan(server, out)
}

func (s *receiverImpl) receiveToChan(server central.SensorService_CommunicateServer, out chan<- *central.MsgFromSensor) {
	// When finished, close input stream so down stream processing ceases gracefully.
	defer s.onFinish()

	for {
		msg, err := server.Recv()
		// Looping stops when the stream closes, or returns an error.
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error("error receiving deployment event: ", err)
			return
		}
		if msg.GetEvent() != nil {
			msg.GetEvent().ClusterId = s.clusterID
		}
		out <- msg
	}
}
