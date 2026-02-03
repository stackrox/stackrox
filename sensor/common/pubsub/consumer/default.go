package consumer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

func newDefaultConsumer(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, callback pubsub.EventCallback) (*DefaultConsumer, error) {
	if callback == nil {
		return nil, errors.Wrap(pubsubErrors.UndefinedEventCallbackErr, "cannot create a consumer with a 'nil' callback")
	}
	return &DefaultConsumer{
		laneID:     laneID,
		topic:      topic,
		consumerID: consumerID,
		callback:   callback,
	}, nil
}

func NewDefaultConsumer(_ ...pubsub.ConsumerOption[*DefaultConsumer]) pubsub.NewConsumer {
	return func(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, callback pubsub.EventCallback) (pubsub.Consumer, error) {
		return newDefaultConsumer(laneID, topic, consumerID, callback)
	}
}

type DefaultConsumer struct {
	laneID     pubsub.LaneID
	topic      pubsub.Topic
	consumerID pubsub.ConsumerID
	callback   pubsub.EventCallback
}

func (c *DefaultConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error)
	go func() {
		defer close(errC)
		start := time.Now()
		operation := metrics.Processed

		select {
		case errC <- func() error {
			err := c.callback(event)
			if err != nil {
				operation = metrics.ConsumerError
			}
			return err
		}():
		case <-waitable.Done():
			operation = metrics.ConsumerError
		}
		metrics.ObserveProcessingDuration(c.laneID, c.topic, c.consumerID, time.Since(start), operation)
		metrics.RecordConsumerOperation(c.laneID, c.topic, c.consumerID, operation)
	}()
	return errC
}

func (c *DefaultConsumer) Stop() {}
