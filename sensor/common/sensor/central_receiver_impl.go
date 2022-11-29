package sensor

import (
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
)

type centralReceiverImpl struct {
	receivers []common.SensorComponent
	stopper   concurrency.Stopper
}

func (s *centralReceiverImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	go s.receive(stream, onStops...)
}

func (s *centralReceiverImpl) Stop(_ error) {
	s.stopper.Client().Stop()
}

func (s *centralReceiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

// Take in data processed by central, run post-processing, then send it to the output channel.
func (s *centralReceiverImpl) receive(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	defer func() {
		s.stopper.Flow().ReportStopped()
		runAll(s.stopper.Client().Stopped().Err(), onStops...)
	}()

	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return

		case <-stream.Context().Done():
			s.stopper.Flow().StopWithError(stream.Context().Err())
			return

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				s.stopper.Flow().StopWithError(nil)
				return
			}
			if err != nil {
				s.stopper.Flow().StopWithError(err)
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
