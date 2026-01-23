package lane

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

type DefaultConfig struct {
	Config
}

func WithDefaultLaneSize(size int) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*defaultLane)
		if !ok {
			panic("cannot use default lane option for this type of lane")
		}
		if size < 0 {
			return
		}
		laneImpl.size = size
	}
}

func WithDefaultLaneConsumer(consumer pubsub.NewConsumer, opts ...pubsub.ConsumerOption) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*defaultLane)
		if !ok {
			panic("cannot use default lane option for this type of lane")
		}
		if consumer == nil {
			panic("cannot configure a 'nil' NewConsumer function")
		}
		laneImpl.newConsumerFn = consumer
		laneImpl.consumerOpts = opts
	}
}

func NewDefaultLane(id pubsub.LaneID, opts ...pubsub.LaneOption) *DefaultConfig {
	return &DefaultConfig{
		Config: Config{
			id:          id,
			opts:        opts,
			newConsumer: consumer.NewDefaultConsumer,
		},
	}
}

func (c *DefaultConfig) NewLane() pubsub.Lane {
	lane := &defaultLane{
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
	lane.ch = make(chan pubsub.Event, lane.size)
	go lane.run()
	return lane
}

type defaultLane struct {
	Lane
	mu      sync.Mutex
	size    int
	ch      chan pubsub.Event
	stopper concurrency.Stopper
}

func (l *defaultLane) Publish(event pubsub.Event) error {
	// We need to lock here and nest two selects to avoid races stopping and
	// publishing events
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.stopper.Flow().StopRequested():
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	default:
	}
	select {
	case <-l.stopper.Flow().StopRequested():
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	case l.ch <- event:
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.Published)
		metrics.SetQueueSize(l.id, len(l.ch))
		return nil
	}
}

func (l *defaultLane) run() {
	defer l.stopper.Flow().ReportStopped()
	for {
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case event, ok := <-l.ch:
			if !ok {
				return
			}
			if err := l.handleEvent(event); err != nil {
				log.Errorf("unable to handle event: %v", err)
			}
		}
	}
}

func (l *defaultLane) handleEvent(event pubsub.Event) error {
	start := time.Now()
	operation := metrics.Processed
	defer func() {
		metrics.ObserveProcessingDuration(l.id, event.Topic(), time.Since(start), operation)
		metrics.SetQueueSize(l.id, len(l.ch))
	}()

	l.consumerLock.RLock()
	defer l.consumerLock.RUnlock()
	consumers, ok := l.consumers[event.Topic()]
	if !ok {
		metrics.RecordConsumerOperation(l.id, event.Topic(), metrics.NoConsumers)
		return errors.Wrap(pubsubErrors.NewConsumersNotFoundForTopicErr(event.Topic(), l.id), "unable to handle event")
	}
	errList := errorhelpers.NewErrorList("handle event")
	for _, c := range consumers {
		select {
		// This will block if we have a slow consumer
		case err := <-c.Consume(l.stopper.Client().Stopped(), event):
			if err != nil {
				errList.AddErrors(pubsubErrors.WrapConsumeErr(err, event.Topic(), l.id))
			}
		case <-l.stopper.Flow().StopRequested():
		}
	}

	if errList.ToError() != nil {
		operation = metrics.ConsumerError
	}
	metrics.RecordConsumerOperation(l.id, event.Topic(), operation)

	return errList.ToError()
}

func (l *defaultLane) RegisterConsumer(topic pubsub.Topic, callback pubsub.EventCallback) error {
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
	metrics.RecordConsumerCount(l.id, topic, len(l.consumers[topic]))
	return nil
}

func (l *defaultLane) Stop() {
	l.stopper.Client().Stop()
	<-l.stopper.Client().Stopped().Done()
	concurrency.WithLock(&l.mu, func() {
		if l.ch == nil {
			return
		}
		close(l.ch)
		l.ch = nil
	})
	l.Lane.Stop()
}
