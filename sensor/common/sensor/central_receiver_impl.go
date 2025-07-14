package sensor

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

var ComponentProcessingDeadline = 5 * time.Second

type centralReceiverImpl struct {
	receivers []common.SensorComponent
	stopper   concurrency.Stopper
	finished  *sync.WaitGroup
}

func (s *centralReceiverImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func()) {
	go s.receive(stream, onStops...)
}

func (s *centralReceiverImpl) Stop() {
	s.stopper.Client().Stop()
}

func (s *centralReceiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

// Take in data processed by central, run post-processing, then send it to the output channel.
func (s *centralReceiverImpl) receive(stream central.SensorService_CommunicateClient, onStops ...func()) {
	defer func() {
		s.stopper.Flow().ReportStopped()
		runAll(onStops...)
		s.finished.Done()
	}()

	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			log.Info("Stop flow requested")
			return

		case <-stream.Context().Done():
			log.Info("Context done")
			s.stopper.Flow().StopWithError(stream.Context().Err())
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
			for _, r := range s.receivers {
				if err := processWithDeadline(r, ComponentProcessingDeadline, msg); err != nil {
					log.Error(err)
				}
			}
		}
	}
}

func processWithDeadline(c common.SensorComponent, deadline time.Duration, msg *central.MsgToSensor) error {
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	// Wrap ProcessMessage with a goroutine
	done := make(chan error)
	defer close(done)
	go func() {
		done <- c.ProcessMessage(msg, ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return errors.Wrapf(ctx.Err(), "component %s took more than %s to process Central reply",
			c.Name(), ComponentProcessingDeadline.String())
	}
}
