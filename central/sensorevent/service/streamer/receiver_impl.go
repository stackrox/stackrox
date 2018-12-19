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
func (s *receiverImpl) Start(stream Stream, out chan<- *central.SensorEvent) {
	go s.receiveToChan(stream, out)
}

func (s *receiverImpl) receiveToChan(stream Stream, out chan<- *central.SensorEvent) {
	// When finished, close input stream so down stream processing ceases gracefully.
	defer s.onFinish()

	for {
		event, err := stream.Recv()
		// Looping stops when the stream closes, or returns an error.
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error("error receiving deployment event: ", err)
			return
		}

		event.ClusterId = s.clusterID
		out <- event
	}
}
