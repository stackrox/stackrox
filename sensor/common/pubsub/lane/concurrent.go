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
	Config[*ConcurrentLane]
}

func WithConcurrentLaneSize(size int) pubsub.LaneOption[*ConcurrentLane] {
	return func(lane *ConcurrentLane) {
		if size < 0 {
			return
		}
		lane.size = size
	}
}

func WithConcurrentLaneConsumer(consumer pubsub.NewConsumer) pubsub.LaneOption[*ConcurrentLane] {
	return func(lane *ConcurrentLane) {
		if consumer == nil {
			panic("cannot configure a 'nil' NewConsumer function")
		}
		lane.newConsumerFn = consumer
	}
}

func NewConcurrentLane(id pubsub.LaneID, opts ...pubsub.LaneOption[*ConcurrentLane]) *ConcurrentConfig {
	return &ConcurrentConfig{
		Config: Config[*ConcurrentLane]{
			id:          id,
			opts:        opts,
			newConsumer: consumer.NewDefaultConsumer(),
		},
	}
}

func (c *ConcurrentConfig) NewLane() pubsub.Lane {
	lane := &ConcurrentLane{
		Lane: Lane{
			id:            c.Config.LaneID(),
			newConsumerFn: c.Config.newConsumer,
			consumers:     make(map[pubsub.Topic][]pubsub.Consumer),
		},
		stopper: concurrency.NewStopper(),
	}
	for _, opt := range c.Config.opts {
		opt(lane)
	}
	lane.ch = safe.NewChannel[pubsub.Event](lane.size, lane.stopper.LowLevel().GetStopRequestSignal())
	go lane.run()
	return lane
}

type ConcurrentLane struct {
	Lane
	size    int
	ch      *safe.Channel[pubsub.Event]
	stopper concurrency.Stopper
}

func (l *ConcurrentLane) Publish(event pubsub.Event) error {
	if err := l.ch.Write(event); err != nil {
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	}
	metrics.RecordPublishOperation(l.id, event.Topic(), metrics.Published)
	metrics.SetQueueSize(l.id, l.ch.Len())
	return nil
}

func (l *ConcurrentLane) run() {
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

func (l *ConcurrentLane) getConsumersByTopic(topic pubsub.Topic) ([]pubsub.Consumer, error) {
	l.consumerLock.RLock()
	defer l.consumerLock.RUnlock()
	consumers, ok := l.consumers[topic]
	if !ok {
		return nil, errors.Wrap(pubsubErrors.NewConsumersNotFoundForTopicErr(topic, l.id), "unable to handle event")
	}
	return consumers, nil
}

func (l *ConcurrentLane) handleEvent(event pubsub.Event) {
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

func (l *ConcurrentLane) handleConsumerError(errC <-chan error) {
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

func (l *ConcurrentLane) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	c, err := l.newConsumerFn(l.id, topic, consumerID, callback)
	if err != nil {
		return errors.Wrap(err, "creating the consumer")
	}
	l.consumerLock.Lock()
	defer l.consumerLock.Unlock()
	l.consumers[topic] = append(l.consumers[topic], c)
	return nil
}

func (l *ConcurrentLane) Stop() {
	l.stopper.Client().Stop()
	// Wait for the run() goroutine to fully exit.
	// The channel will be closed automatically when the waitable is done.
	<-l.stopper.Client().Stopped().Done()
	l.Lane.Stop()
}
