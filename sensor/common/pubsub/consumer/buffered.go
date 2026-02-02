package consumer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/channel"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
)

// bufferedEvent wraps an event with its error channel to pipe callback errors back to the caller
type bufferedEvent struct {
	event pubsub.Event
	errC  chan error
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

func NewBufferedConsumer(callback pubsub.EventCallback, opts ...pubsub.ConsumerOption) (pubsub.Consumer, error) {
	if callback == nil {
		return nil, errors.Wrap(pubsubErrors.UndefinedEventCallbackErr, "")
	}
	ret := &BufferedConsumer{
		callback: callback,
		stopper:  concurrency.NewStopper(),
		size:     1000,
	}
	for _, opt := range opts {
		opt(ret)
	}
	ret.buffer = channel.NewSafeChannel[*bufferedEvent](ret.size, ret.stopper.LowLevel().GetStopRequestSignal())
	go ret.run()
	return ret, nil
}

type BufferedConsumer struct {
	callback pubsub.EventCallback
	size     int
	stopper  concurrency.Stopper
	buffer   *channel.SafeChannel[*bufferedEvent]
}

func (c *BufferedConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error, 1)
	go func() {
		// Priority 1: Check if already cancelled
		select {
		case <-waitable.Done():
			close(errC)
			return
		case <-c.stopper.Flow().StopRequested():
			close(errC)
			return
		default:
		}

		// Wrap event with its errC to pipe callback errors back to caller
		wrappedEvent := &bufferedEvent{
			event: event,
			errC:  errC,
		}

		// SafeChannel.TryWrite is non-blocking by design, so it's safe to call directly
		writeErr := c.buffer.TryWrite(wrappedEvent)

		// Priority 2: If write failed, send error and close. Otherwise keep errC open.
		if writeErr != nil {
			select {
			case errC <- writeErr:
				close(errC)
			case <-waitable.Done():
				close(errC)
			case <-c.stopper.Flow().StopRequested():
				close(errC)
			}
		}
		// If writeErr is nil, errC stays open and will be closed later when callback completes
	}()
	return errC
}

func (c *BufferedConsumer) Stop() {
	c.stopper.Client().Stop()
	<-c.stopper.Client().Stopped().Done()
	c.buffer.Close()
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
			// Execute callback in separate goroutine to prevent blocking the consumer
			callbackDone := make(chan error, 1)
			go func() {
				callbackDone <- c.callback(wrappedEv.event)
				close(callbackDone)
			}()
			// Wait for callback or stopper, allowing clean exit if callback blocks
			select {
			case <-c.stopper.Flow().StopRequested():
				// Consumer is stopping - close the errC without waiting for callback
				close(wrappedEv.errC)
				return
			case err := <-callbackDone:
				// Callback completed - send result (nil or error) and close errC
				wrappedEv.errC <- err
				close(wrappedEv.errC)
			}
		}
	}
}
