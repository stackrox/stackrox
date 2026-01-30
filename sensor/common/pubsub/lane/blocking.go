package lane

import (
	"github.com/pkg/errors"

	"github.com/stackrox/rox/pkg/channel"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

type BlockingConfig struct {
	Config
}

func WithBlockingLaneSize(size int) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*blockingLane)
		if !ok {
			panic("cannot use default lane option for this type of lane")
		}
		if size < 0 {
			return
		}
		laneImpl.size = size
	}
}

func WithBlockingLaneConsumer(consumer pubsub.NewConsumer, opts ...pubsub.ConsumerOption) pubsub.LaneOption {
	return func(lane pubsub.Lane) {
		laneImpl, ok := lane.(*blockingLane)
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

func NewBlockingLane(id pubsub.LaneID, opts ...pubsub.LaneOption) *BlockingConfig {
	return &BlockingConfig{
		Config: Config{
			id:          id,
			opts:        opts,
			newConsumer: consumer.NewDefaultConsumer,
		},
	}
}

func (c *BlockingConfig) NewLane() pubsub.Lane {
	lane := &blockingLane{
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
	lane.ch = channel.NewSafeChannel[pubsub.Event](lane.size, lane.stopper.LowLevel().GetStopRequestSignal())
	go lane.run()
	return lane
}

type blockingLane struct {
	Lane
	size    int
	ch      *channel.SafeChannel[pubsub.Event]
	stopper concurrency.Stopper
}

func (l *blockingLane) Publish(event pubsub.Event) error {
	if err := l.ch.Write(event); err != nil {
		metrics.RecordPublishOperation(l.id, event.Topic(), metrics.PublishError)
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	}
	metrics.RecordPublishOperation(l.id, event.Topic(), metrics.Published)
	metrics.SetQueueSize(l.id, l.ch.Len())
	return nil
}

func (l *blockingLane) run() {
	defer l.stopper.Flow().ReportStopped()
	for {
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case event, ok := <-l.ch.Chan():
			if !ok {
				return
			}
			if err := l.handleEvent(event); err != nil {
				log.Errorf("unable to handle event: %v", err)
			}
		}
	}
}

func (l *blockingLane) handleEvent(event pubsub.Event) error {
	defer func() {
		metrics.SetQueueSize(l.id, l.ch.Len())
	}()

	l.consumerLock.RLock()
	defer l.consumerLock.RUnlock()
	consumers, ok := l.consumers[event.Topic()]
	if !ok {
		metrics.RecordConsumerOperation(l.id, event.Topic(), pubsub.NoConsumers, metrics.NoConsumers)
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

	return errList.ToError()
}

func (l *blockingLane) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
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
	metrics.RecordConsumerCount(l.id, topic, len(l.consumers[topic]))
	return nil
}

func (l *blockingLane) Stop() {
	l.stopper.Client().Stop()
	l.ch.Close()
	l.Lane.Stop()
}
