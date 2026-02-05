package lane

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/safe"
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
			panic("attempted using concurrent lane option for a different lane type")
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
			panic("attempted using concurrent lane option for a different lane type")
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
		stopper: concurrency.NewStopper(),
	}
	for _, opt := range c.opts {
		opt(lane)
	}
	lane.ch = safe.NewChannel[pubsub.Event](lane.size, lane.stopper.LowLevel().GetStopRequestSignal())
	go lane.run()
	return lane
}

type concurrentLane struct {
	Lane
	size    int
	ch      *safe.Channel[pubsub.Event]
	stopper concurrency.Stopper
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
		// Priority 1: Check if stop requested
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		default:
		}
		// Priority 2: Read event, but respect stop during blocking read
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
		log.Errorf("unable to handle event: %v", err)
		metrics.RecordConsumerOperation(l.id, event.Topic(), pubsub.NoConsumers, metrics.NoConsumers)
		return
	}
	for _, c := range consumers {
		errC := c.Consume(l.stopper.Client().Stopped(), event)
		// Spawning go routine here to not block other consumers
		go l.handleConsumerError(errC)
	}
}

func (l *concurrentLane) handleConsumerError(errC <-chan error) {
	// This blocks until the consumer finishes the processing
	// TODO: Consider adding a timeout here
	select {
	case err := <-errC:
		if err != nil {
			// TODO: consider adding a callback to inform of the error
			log.Errorf("unable to handle event: %v", err)
		}
	case <-l.stopper.Flow().StopRequested():
	}
}

func (l *concurrentLane) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	c, err := l.newConsumerFn(l.id, topic, consumerID, callback, l.consumerOpts...)
	if err != nil {
		return errors.Wrap(err, "creating the consumer")
	}
	l.consumerLock.Lock()
	defer l.consumerLock.Unlock()
	l.consumers[topic] = append(l.consumers[topic], c)
	return nil
}

func (l *concurrentLane) Stop() {
	l.stopper.Client().Stop()
	// Wait for the run() goroutine to fully exit before closing the channel.
	// This ensures an orderly shutdown where event processing is complete.
	<-l.stopper.Client().Stopped().Done()
	l.ch.Close()
	l.Lane.Stop()
}
