package consumer

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

func NewDefaultConsumer(callback pubsub.EventCallback, _ ...pubsub.ConsumerOption) pubsub.Consumer {
	return &DefaultConsumer{
		Consumer: Consumer{
			callback: callback,
		},
	}
}

type DefaultConsumer struct {
	Consumer
}

func (c *DefaultConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error)
	if !c.isCallbackConfigured(waitable, errC) {
		return errC
	}
	go func() {
		defer close(errC)
		select {
		case errC <- c.callback(event):
		case <-waitable.Done():
		}
	}()
	return errC
}

func (c *DefaultConsumer) Stop() {}
