package sensor

import (
	"context"
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/metrics"
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
		// Initialize queue size metric to 0
		metrics.SetCentralReceiverComponentQueueSize(r.Name(), 0)
	}

	wg := sync.WaitGroup{}

	// Start periodic queue size updates
	queueSizeTicker := time.NewTicker(5 * time.Second)
	defer queueSizeTicker.Stop()
	go func() {
		for {
			select {
			case <-queueSizeTicker.C:
				for componentName, ch := range componentsQueues {
					metrics.SetCentralReceiverComponentQueueSize(componentName, len(ch))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		wg.Wait()
		go func() {
			for name, ch := range componentsQueues {
				for msg := range ch {
					log.Warnf("Dropping %s not handled by %s", msg.String(), name)
					metrics.IncrementCentralReceiverMessagesDropped(name, "shutdown")
				}
			}
		}()
		cancel()
		for name, ch := range componentsQueues {
			log.Debugf("Closing component queue %s", name)
			close(ch)
		}

		s.stopper.Flow().ReportStopped()
		runAll(onStops...)
		s.finished.Done()
	}()

	for _, receiver := range s.receivers {
		go process(ctx, componentsQueues[receiver.Name()], receiver)
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
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	localWg := &sync.WaitGroup{}
	wg.Add(len(componentsQueues))
	localWg.Add(len(componentsQueues))
	go func() {
		localWg.Wait()
		cancel()
	}()
	for name, ch := range componentsQueues {
		go func(componentName string, ch chan *central.MsgToSensor) {
			defer func() {
				localWg.Done()
				wg.Done()
			}()
			sendStart := time.Now()
			select {
			case <-ctx.Done():
				log.Infof("Context %s, not multiplexing messages. Dropping %s", ctx.Err(), msg.String())
				metrics.IncrementCentralReceiverMessagesDropped(componentName, "timeout")
				return
			case ch <- msg:
				metrics.ObserveCentralReceiverChannelSendDuration(componentName, time.Since(sendStart))
			}
		}(name, ch)
	}
}

func process(ctx context.Context, ch <-chan *central.MsgToSensor, r common.SensorComponent) {
	for msg := range ch {
		if ctx.Err() != nil {
			metrics.IncrementCentralReceiverMessagesDropped(r.Name(), "shutdown")
			return
		}
		start := time.Now()
		if err := r.ProcessMessage(ctx, msg); err != nil {
			log.Errorf("%s: %+v", r.Name(), err)
		}
		metrics.ObserveCentralReceiverProcessMessageDuration(r.Name(), time.Since(start))
	}
}
