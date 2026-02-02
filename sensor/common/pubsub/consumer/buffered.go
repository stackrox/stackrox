package consumer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/channel"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
)

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
	ret.buffer = channel.NewSafeChannel[pubsub.Event](ret.size, ret.stopper.LowLevel().GetStopRequestSignal())
	go ret.run()
	return ret, nil
}

type BufferedConsumer struct {
	callback pubsub.EventCallback
	size     int
	stopper  concurrency.Stopper
	buffer   *channel.SafeChannel[pubsub.Event]
}

func (c *BufferedConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		// Priority 1: Check if already cancelled
		select {
		case <-waitable.Done():
			return
		case <-c.stopper.Flow().StopRequested():
			return
		default:
		}
		// Priority 2: Try to write, respecting cancellation during the write
		select {
		case errC <- c.buffer.TryWrite(event):
		case <-waitable.Done():
		case <-c.stopper.Flow().StopRequested():
		}
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
		case ev, ok := <-c.buffer.Chan():
			if !ok {
				return
			}
			// Execute callback in separate goroutine to prevent blocking the consumer
			callbackDone := make(chan error, 1)
			go func() {
				callbackDone <- c.callback(ev)
			}()
			// Wait for callback or stopper, allowing clean exit if callback blocks
			select {
			case <-c.stopper.Flow().StopRequested():
				return
			case err := <-callbackDone:
				if err != nil {
					// TODO: Pipe error to errC Created in Consume
				}
			}
		}
	}
}
