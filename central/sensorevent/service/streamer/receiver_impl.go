package streamer

import (
	"io"

	"github.com/stackrox/rox/generated/api/v1"
)

type receiverImpl struct {
	clusterID string
	onFinish  func()
}

// Start starts receiving from the grpc stream and pushing recieved events to the out channel.
func (s *receiverImpl) Start(stream v1.SensorEventService_RecordEventServer, out chan<- *v1.SensorEvent) {
	go s.receiveToChan(stream, out)
}

func (s *receiverImpl) receiveToChan(stream v1.SensorEventService_RecordEventServer, out chan<- *v1.SensorEvent) {
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
