package consumer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/utils"
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

type eventWithErrC struct {
	event pubsub.Event
	errC  chan<- error
}

func NewBufferedConsumer(callback pubsub.EventCallback, opts ...pubsub.ConsumerOption) (pubsub.Consumer, error) {
	if callback == nil {
		return nil, errors.Wrap(pubsubErrors.UndefinedEventCallbackErr, "")
	}
	ret := &BufferedConsumer{
		callback: callback,
		stopper:  concurrency.NewStopper(),
		size:     10,
	}
	for _, opt := range opts {
		opt(ret)
	}
	ret.buffer = make(chan *eventWithErrC, ret.size)
	go ret.run()
	return ret, nil
}

type BufferedConsumer struct {
	callback pubsub.EventCallback
	mu       sync.Mutex
	buffer   chan *eventWithErrC
	size     int
	stopper  concurrency.Stopper
}

func (c *BufferedConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error)
	eventErrC := make(chan error)
	go func() {
		defer close(errC)
		err := c.consume(event, eventErrC)
		if err != nil {
			select {
			case errC <- err:
			case <-waitable.Done():
			case <-c.stopper.Flow().StopRequested():
			}
			return
		}
		select {
		case eventErr := <-eventErrC:
			select {
			case errC <- eventErr:
			case <-waitable.Done():
			case <-c.stopper.Flow().StopRequested():
			}
		case <-waitable.Done():
		case <-c.stopper.Flow().StopRequested():
		}
	}()
	return errC
}

func (c *BufferedConsumer) consume(event pubsub.Event, eventErrC chan<- error) error {
	if err := utils.SafeWriteToChannel[*eventWithErrC](
		&c.mu,
		c.stopper.LowLevel().GetStopRequestSignal(),
		c.buffer,
		&eventWithErrC{event: event, errC: eventErrC},
	); err != nil {
		if errors.Is(err, pubsubErrors.ChannelFullErr) {
			return errors.Wrap(pubsubErrors.NewConsumerBufferFullError(event.Topic(), event.Lane()), "unable to handle event")
		}
		return errors.Wrap(pubsubErrors.NewConsumeOnStoppedConsumerErr(event.Topic(), event.Lane()), "unable to handle event")
	}
	return nil
}

func (c *BufferedConsumer) Stop() {
	c.stopper.Client().Stop()
	<-c.stopper.Client().Stopped().Done()
	concurrency.WithLock(&c.mu, func() {
		if c.buffer != nil {
			close(c.buffer)
		}
		c.buffer = nil
	})
}

func (c *BufferedConsumer) run() {
	defer c.stopper.Flow().ReportStopped()
	for {
		select {
		case <-c.stopper.Flow().StopRequested():
			return
		case ev, ok := <-c.buffer:
			if !ok {
				return
			}
			select {
			case ev.errC <- c.callback(ev.event):
			case <-c.stopper.Flow().StopRequested():
			}
			close(ev.errC)
		}
	}
}
