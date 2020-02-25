package sensor

import (
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
)

type centralReceiverImpl struct {
	receivers []common.SensorComponent

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (s *centralReceiverImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	go s.receive(stream, onStops...)
}

func (s *centralReceiverImpl) Stop(err error) {
	s.stopC.SignalWithError(err)
}

func (s *centralReceiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *centralReceiverImpl) receive(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		runAll(s.stopC.Err(), onStops...)
	}()

	for {
		select {
		case <-s.stopC.Done():
			return

		case <-stream.Context().Done():
			s.stopC.SignalWithError(stream.Context().Err())
			return

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				s.stopC.Signal()
				return
			}
			if err != nil {
				s.stopC.SignalWithError(err)
				return
			}
			for _, r := range s.receivers {
				if err := r.ProcessMessage(msg); err != nil {
					log.Error(err)
				}
			}
		}
	}
}
