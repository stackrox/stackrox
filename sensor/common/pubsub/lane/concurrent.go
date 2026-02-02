package lane

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/channel"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
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
			newConsumer: consumer.NewDefaultConsumer,
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
	}
	for _, opt := range c.opts {
		opt(lane)
	}
	lane.ch = channel.NewSafeChannel[pubsub.Event](lane.size, lane.stopper.LowLevel().GetStopRequestSignal())
	lane.errC = channel.NewSafeChannel[error](0, lane.stopper.LowLevel().GetStopRequestSignal())
	go lane.run()
	go lane.runHandleErr()
	return lane
}

type concurrentLane struct {
	Lane
	size                  int
	ch                    *channel.SafeChannel[pubsub.Event]
	errC                  *channel.SafeChannel[error]
	stopper               concurrency.Stopper
	errHandlingStopSignal concurrency.Signal
}

func (l *concurrentLane) Publish(event pubsub.Event) error {
	if err := l.ch.Write(event); err != nil {
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	}
	metrics.RecordPublishOperation(l.id, event.Topic(), metrics.Published)
	metrics.SetQueueSize(l.id, l.ch.Len())
	return nil
}

func (l *concurrentLane) run() {
	defer l.stopper.Flow().ReportStopped()
	for {
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case event, ok := <-l.ch.Chan():
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
		case err, ok := <-l.errC.Chan():
			if !ok {
				return
			}
			// TODO: consider adding a callback to inform of the error
			log.Errorf("unable to handle event: %v", err)
		}
	}
}

func (l *concurrentLane) getConsumersByTopic(topic pubsub.Topic) ([]pubsub.Consumer, error) {
	l.consumerLock.RLock()
	defer l.consumerLock.RUnlock()
	consumers, ok := l.consumers[topic]
	if !ok {
		return nil, errors.Wrap(pubsubErrors.NewConsumersNotFoundForTopicErr(topic, l.id), "unable to handle event")
	}
	return consumers, nil
}

func (l *concurrentLane) handleEvent(event pubsub.Event) {
	defer metrics.SetQueueSize(l.id, l.ch.Len())
	consumers, err := l.getConsumersByTopic(event.Topic())
	if err != nil {
		l.writeToErrChannel(err)
		metrics.RecordConsumerOperation(l.id, event.Topic(), pubsub.NoConsumers, metrics.NoConsumers)
		return
	}
	for _, c := range consumers {
		errC := c.Consume(l.stopper.Client().Stopped(), event)
		// Spawning go routine here to not block other consumers
		go func() {
			// This blocks until the consumer finishes the processing
			// TODO: Consider adding a timout here
			select {
			case err := <-errC:
				// write to channel does nothing if err == nil
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
	if err := l.errC.Write(err); err != nil {
		// This is ok. We should only fail to write to the channel if sensor is stopping
		log.Warn("unable to write consumer error to error channel")
	}
}

func (l *concurrentLane) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	c, err := l.newConsumerFn(l.id, topic, consumerID, callback, l.consumerOpts...)
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
	l.ch.Close()
	l.errC.Close()
	<-l.errHandlingStopSignal.Done()
	l.Lane.Stop()
}
