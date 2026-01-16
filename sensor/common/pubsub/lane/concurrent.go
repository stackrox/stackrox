package lane

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub/utils"
)

type ConcurrentConfig struct {
	Config
}

func WithConcurrentLaneSize(size int) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*concurrentLane)
		if !ok {
			panic("cannot use concurrent lane option for this type of lane")
		}
		if size < 0 {
			return
		}
		laneImpl.size = size
	}
}

func WithConcurrentLaneConsumer(consumer pubsub.NewConsumer, opts ...pubsub.ConsumerOption) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*concurrentLane)
		if !ok {
			panic("cannot use concurrent lane option for this type of lane")
		}
		if consumer == nil {
			panic("cannot configure a 'nil' NewConsumer function")
		}
		laneImpl.newConsumerFn = consumer
		laneImpl.consumerOpts = opts
	}
}

func NewConcurrentLane(id pubsub.LaneID, opts ...pubsub.LaneOption) *ConcurrentConfig {
	return &ConcurrentConfig{
		Config: Config{
			id:          id,
			opts:        opts,
			newConsumer: consumer.NewBufferedConsumer,
		},
	}
}

func (c *ConcurrentConfig) NewLane() pubsub.Lane {
	lane := &concurrentLane{
		Lane: Lane{
			id:            c.LaneID(),
			newConsumerFn: c.newConsumer,
			consumers:     make(map[pubsub.Topic][]pubsub.Consumer),
		},
		stopper:               concurrency.NewStopper(),
		errHandlingStopSignal: concurrency.NewSignal(),
		errC:                  make(chan error),
	}
	for _, opt := range c.opts {
		opt(lane)
	}
	lane.ch = make(chan pubsub.Event, lane.size)
	go lane.run()
	go lane.runHandleErr()
	return lane
}

type concurrentLane struct {
	Lane
	chLock                sync.Mutex
	errCLock              sync.Mutex
	size                  int
	ch                    chan pubsub.Event
	errC                  chan error
	stopper               concurrency.Stopper
	errHandlingStopSignal concurrency.Signal
}

func (l *concurrentLane) Publish(event pubsub.Event) error {
	if err := utils.SafeBlockingWriteToChannel[pubsub.Event](&l.chLock, l.stopper.LowLevel().GetStopRequestSignal(), l.ch, event); err != nil {
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	}
	metrics.RecordPublishOperation(l.id, event.Topic(), metrics.Published)
	metrics.SetQueueSize(l.id, len(l.ch))
	return nil
}

func (l *concurrentLane) run() {
	defer l.stopper.Flow().ReportStopped()
	for {
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case event, ok := <-l.ch:
			if !ok {
				return
			}
			l.handleEvent(event)
		}
	}
}

func (l *concurrentLane) runHandleErr() {
	defer l.errHandlingStopSignal.Signal()
	for {
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case err, ok := <-l.errC:
			if !ok {
				return
			}
			// TODO: consider adding a callback to inform of the error
			log.Errorf("unable to handle event: %v", err)
		}
	}
}

func (l *concurrentLane) handleEvent(event pubsub.Event) {
	start := time.Now()
	operation := metrics.Processed
	finishedProcessingSignal := concurrency.NewSignal()
	var consumerError atomic.Bool
	consumerError.Store(false)
	defer func() {
		select {
		case <-l.stopper.Flow().StopRequested():
		case <-finishedProcessingSignal.Done():
		}
		metrics.ObserveProcessingDuration(l.id, event.Topic(), time.Since(start), operation)
		metrics.SetQueueSize(l.id, len(l.ch))
		if consumerError.Load() {
			operation = metrics.ConsumerError
		}
		metrics.RecordConsumerOperation(l.id, event.Topic(), operation)
	}()
	var consumers []pubsub.Consumer
	if err := concurrency.WithLock1[error](&l.consumerLock, func() error {
		var ok bool
		consumers, ok = l.consumers[event.Topic()]
		if !ok {
			return errors.Wrap(pubsubErrors.NewConsumersNotFoundForTopicErr(event.Topic(), l.id), "unable to handle event")
		}
		return nil
	}); err != nil {
		operation = metrics.NoConsumers
		l.writeToErrChannel(err)
		finishedProcessingSignal.Signal()
		return
	}
	var waitForConsumers atomic.Int32
	waitForConsumers.Store(int32(len(consumers)))
	for _, c := range consumers {
		errC := c.Consume(l.stopper.Client().Stopped(), event)
		// Spawning go routine here to not block other consumers
		go func() {
			defer func() {
				waitForConsumers.Add(-1)
				if waitForConsumers.Load() == 0 {
					finishedProcessingSignal.Signal()
				}
			}()
			// This blocks until the consumer finishes the processing
			// TODO: Consider adding a timout here
			select {
			case err := <-errC:
				consumerError.Store(true)
				l.writeToErrChannel(err)
			case <-l.stopper.Flow().StopRequested():
			}
		}()
	}
}

func (l *concurrentLane) writeToErrChannel(err error) {
	if err == nil {
		return
	}
	if err := utils.SafeBlockingWriteToChannel[error](&l.errCLock, l.stopper.LowLevel().GetStopRequestSignal(), l.errC, err); err != nil {
		// This is ok. We should only fail to write to the channel if sensor is stopping
		log.Warn("unable to write consumer error to error channel")
	}
}

func (l *concurrentLane) RegisterConsumer(topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	c, err := l.newConsumerFn(callback, l.consumerOpts...)
	if err != nil {
		return errors.Wrap(err, "unable to create the consumer")
	}
	l.consumerLock.Lock()
	defer l.consumerLock.Unlock()
	l.consumers[topic] = append(l.consumers[topic], c)
	return nil
}

func (l *concurrentLane) Stop() {
	l.stopper.Client().Stop()
	<-l.stopper.Client().Stopped().Done()
	<-l.errHandlingStopSignal.Done()
	concurrency.WithLock(&l.chLock, func() {
		if l.ch != nil {
			close(l.ch)
			l.ch = nil
		}
	})
	concurrency.WithLock(&l.errCLock, func() {
		if l.errC != nil {
			close(l.errC)
			l.errC = nil
		}
	})
	l.Lane.Stop()
}
