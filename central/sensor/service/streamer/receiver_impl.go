package streamer

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

type receiverImpl struct {
	clusterID string

	output chan *central.MsgFromSensor

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

// Start starts receiving from the grpc stream and pushing recieved events to the out channel.
func (s *receiverImpl) Start(server central.SensorService_CommunicateServer, dependents ...Stoppable) {
	go s.receiveToChan(server, dependents...)
}

func (s *receiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *receiverImpl) Output() <-chan *central.MsgFromSensor {
	return s.output
}

func (s *receiverImpl) receiveToChan(server central.SensorService_CommunicateServer, dependents ...Stoppable) {
	defer func() {
		StopAll(s.stoppedC.Err(), dependents...)
	}()

	for {
		msg, err := server.Recv()
		// Looping stops when the stream closes, or returns an error.
		if err == io.EOF {
			s.stoppedC.Signal()
			return
		}
		if err != nil {
			s.stoppedC.SignalWithError(err)
			return
		}
		if !s.writeToOutput(msg) {
			log.Debugf("message received from sensor dropped: %s", proto.MarshalTextString(msg))
		}
	}
}

func (s *receiverImpl) writeToOutput(out *central.MsgFromSensor) bool {
	select {
	case s.output <- out:
		return true
	case <-s.stoppedC.Done():
		return false
	}
}
