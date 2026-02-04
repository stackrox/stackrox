package consumer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

const (
	defaultBufferSize = 1000
)

// bufferedEvent wraps an event with its error channel to pipe callback errors back to the caller
type bufferedEvent struct {
	event     pubsub.Event
	errC      chan<- error
	startTime time.Time
}

func WithBufferedConsumerSize(size int) pubsub.ConsumerOption {
	return func(consumer pubsub.Consumer) {
		impl, ok := consumer.(*BufferedConsumer)
		if !ok {
			return
		}
		if size < 0 {
			return
		}
		impl.size = size
	}
}

func NewBufferedConsumer(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, callback pubsub.EventCallback, opts ...pubsub.ConsumerOption) (pubsub.Consumer, error) {
	if callback == nil {
		return nil, errors.Wrap(pubsubErrors.UndefinedEventCallbackErr, "")
	}
	ret := &BufferedConsumer{
		laneID:     laneID,
		topic:      topic,
		consumerID: consumerID,
		callback:   callback,
		stopper:    concurrency.NewStopper(),
		size:       defaultBufferSize,
	}
	for _, opt := range opts {
		opt(ret)
	}
	ret.buffer = safe.NewChannel[*bufferedEvent](ret.size, ret.stopper.LowLevel().GetStopRequestSignal())
	go ret.run()
	return ret, nil
}

type BufferedConsumer struct {
	laneID     pubsub.LaneID
	topic      pubsub.Topic
	consumerID pubsub.ConsumerID
	callback   pubsub.EventCallback
	size       int
	stopper    concurrency.Stopper
	buffer     *safe.Channel[*bufferedEvent]
}

func (c *BufferedConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error, 1)
	// No goroutine needed: all operations in consume are non-blocking.
	// The select statements use default cases, TryWrite is non-blocking by design,
	// and errC has size 1 so the single send on error won't block.
	c.consume(waitable, event, errC)
	return errC
}

func (c *BufferedConsumer) consume(waitable concurrency.Waitable, event pubsub.Event, errC chan<- error) {
	// IMPORTANT: All operations must remain non-blocking.
	start := time.Now()
	operation := metrics.ConsumerError

	// Priority 1: Check if already cancelled
	select {
	case <-waitable.Done():
		close(errC)
		c.recordMetrics(operation, start)
		return
	case <-c.stopper.Flow().StopRequested():
		close(errC)
		c.recordMetrics(operation, start)
		return
	default:
	}

	// Wrap event with its errC to pipe callback errors back to caller
	wrappedEvent := &bufferedEvent{
		event:     event,
		errC:      errC,
		startTime: start,
	}

	// SafeChannel.TryWrite is non-blocking by design, so it's safe to call directly
	writeErr := c.buffer.TryWrite(wrappedEvent)

	// Priority 2: If write failed, send error and close. Otherwise keep errC open.
	if writeErr != nil {
		operation := metrics.ConsumerError
		errC <- writeErr // Won't block - buffered channel of size 1
		close(errC)
		c.recordMetrics(operation, start)
	}
	// If writeErr is nil, errC stays open and will be closed later when callback completes
}

func (c *BufferedConsumer) Stop() {
	c.stopper.Client().Stop()
	<-c.stopper.Client().Stopped().Done()
	c.buffer.Close()
	// Drain events and close their errC
	for ev := range c.buffer.Chan() {
		close(ev.errC)
	}
}

func (c *BufferedConsumer) run() {
	defer c.stopper.Flow().ReportStopped()
	for {
		// Priority 1: Check if stop requested
		select {
		case <-c.stopper.Flow().StopRequested():
			return
		default:
		}
		// Priority 2: Read event, but respect stop during blocking read
		select {
		case <-c.stopper.Flow().StopRequested():
			return
		case wrappedEv, ok := <-c.buffer.Chan():
			if !ok {
				return
			}
			c.handleEvent(wrappedEv)
		}
	}
}

func (c *BufferedConsumer) handleEvent(wrappedEv *bufferedEvent) {
	// Execute callback in separate goroutine to prevent blocking the consumer
	callbackDone := make(chan error, 1)
	go func() {
		callbackDone <- c.callback(wrappedEv.event)
		close(callbackDone)
	}()
	// Wait for callback or stopper, allowing clean exit if callback blocks
	operation := metrics.Processed
	select {
	case <-c.stopper.Flow().StopRequested():
		// Consumer is stopping - close the errC without waiting for callback
		operation = metrics.ConsumerError
		close(wrappedEv.errC)
		c.recordMetrics(operation, wrappedEv.startTime)
		return
	case err := <-callbackDone:
		// Callback completed - send result (nil or error) and close errC
		if err != nil {
			operation = metrics.ConsumerError
		}
		wrappedEv.errC <- err
		close(wrappedEv.errC)
		c.recordMetrics(operation, wrappedEv.startTime)
	}
}

func (c *BufferedConsumer) recordMetrics(op metrics.Operation, start time.Time) {
	metrics.ObserveProcessingDuration(c.laneID, c.topic, c.consumerID, time.Since(start), op)
	metrics.RecordConsumerOperation(c.laneID, c.topic, c.consumerID, op)
}
