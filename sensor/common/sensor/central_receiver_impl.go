package sensor

import (
	"context"
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

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
	ctx, cancel := context.WithCancel(stream.Context())

	componentsQueues := make(map[string]chan *central.MsgToSensor, len(s.receivers))
	for _, r := range s.receivers {
		componentsQueues[r.Name()] = make(chan *central.MsgToSensor, 1)
	}

	defer func() {
		cancel()
		s.stopper.Flow().ReportStopped()
		runAll(onStops...)
		s.finished.Done()
		for name, ch := range componentsQueues {
			go func() {
				for msg := range ch {
					log.Warnf("Dropping %s not handled by %s", msg, name)
				}
			}()
		}
		for name, ch := range componentsQueues {
			log.Debug("Closing component queue %s", name)
			close(ch)
		}
	}()

	sendToAll := func(ctx context.Context, msg *central.MsgToSensor) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		for name, ch := range componentsQueues {
			select {
			case ch <- msg:
				log.Debugf("Sending msg to %s", name)
			case <-ctx.Done():
				log.Errorf("Failed to send msg %T to %s receiver channel", msg, name)
			}
		}
	}

	for _, receiver := range s.receivers {
		go process(componentsQueues[receiver.Name()], receiver, receiver.Name())
	}

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
			go sendToAll(ctx, msg)
		}
	}
}

func process(ch <-chan *central.MsgToSensor, r common.CentralReceiver, name string) {
	for msg := range ch {
		if err := r.ProcessMessage(msg); err != nil {
			log.Errorf("%s: %+v", name, err)
		}
	}
	log.Infof("Stopping %s", name)
}
