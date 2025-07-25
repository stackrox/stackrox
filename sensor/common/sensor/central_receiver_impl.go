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
		componentsQueues[r.Name()] = make(chan *central.MsgToSensor, 10)
	}

	wg := sync.WaitGroup{}

	defer func() {
		log.Info("Waiting for processors")
		wg.Wait()
		for name, ch := range componentsQueues {
			go func() {
				for msg := range ch {
					log.Warnf("Dropping %s not handled by %s", msg.String(), name)
				}
			}()
		}
		log.Info("Canceling context")
		cancel()
		for name, ch := range componentsQueues {
			log.Infof("Closing component queue %s", name)
			close(ch)
		}

		log.Info("Reporting Stopped")
		s.stopper.Flow().ReportStopped()
		log.Info("Run All on Stops")
		runAll(onStops...)
		log.Info("Finished DONE")
		s.finished.Done()
	}()

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
			sendToAll(ctx, msg, &wg, componentsQueues)
		}
	}
}

func sendToAll(ctx context.Context, msg *central.MsgToSensor, wg *sync.WaitGroup, componentsQueues map[string]chan *central.MsgToSensor) {
	wg.Add(len(componentsQueues))
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	for name, ch := range componentsQueues {
		go func() {
			defer func() {
				wg.Done()
			}()
			select {
			case <-ctx.Done():
				log.Infof("Context %s, not multiplexing messages. Dropping %s", ctx.Err(), msg.String())
				return
			case ch <- msg:
				log.Infof("Sending msg to %s", name)
			}
		}()
	}
}

func process(ch <-chan *central.MsgToSensor, r common.CentralReceiver, name string) {
	for msg := range ch {
		if err := r.ProcessMessage(msg); err != nil {
			log.Errorf("%s: %+v", name, err)
		}
	}
}
