package sensor

import (
	"context"
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type centralReceiverImpl struct {
	processor *ComponentProcessor
	stopper   concurrency.Stopper
	finished  *sync.WaitGroup
}

func (s *centralReceiverImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func()) {
	go s.receive(stream, onStops...)
}

func (s *centralReceiverImpl) Stop() {
	log.Debug("Stopping CentralReceiver")
	s.stopper.Client().Stop()
}

func (s *centralReceiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

// Take in data processed by central, run post-processing, then send it to the output channel.
func (s *centralReceiverImpl) receive(stream central.SensorService_CommunicateClient, onStops ...func()) {
	ctx, cancel := context.WithCancel(stream.Context())

	defer func() {
		cancel()
		s.stopper.Flow().ReportStopped()
		runAll(onStops...)
		s.finished.Done()
	}()

	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			log.Info("Stop flow requested")
			return

		case <-ctx.Done():
			log.Info("Context done")
			s.stopper.Flow().StopWithError(ctx.Err())
			return

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				log.Info("EOF on gRPC stream")
				s.stopper.Flow().StopWithError(nil)
				return
			}
			if err != nil {
				log.Infof("Stopping with error: %s", err)
				s.stopper.Flow().StopWithError(err)
				return
			}
			s.processor.ProcessMessage(msg)
		}
	}
}
