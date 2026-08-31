package consumer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

const defaultConsumerLoggingRateLimiter = "pubsub-default-consumer"

func NewDefaultConsumer() pubsub.NewConsumer {
	return func(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, callback pubsub.EventCallback) (pubsub.Consumer, error) {
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
}

type DefaultConsumer struct {
	laneID     pubsub.LaneID
	topic      pubsub.Topic
	consumerID pubsub.ConsumerID
	callback   pubsub.EventCallback
}

func (c *DefaultConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	// Buffered so the send below never blocks on a reader: callers (e.g. concurrentLane)
	// are not required to drain this channel, and must not need to in order for this
	// goroutine to exit.
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		start := time.Now()
		operation := metrics.Processed

		select {
		case errC <- func() error {
			err := c.callback(event)
			if err != nil {
				operation = metrics.ConsumerError
				logging.GetRateLimitedLogger().ErrorL(defaultConsumerLoggingRateLimiter, "unable to handle event on lane %s, topic %s: %v", c.laneID, c.topic, err)
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
