package consumer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
)

func NewDefaultConsumer(callback pubsub.EventCallback, _ ...pubsub.ConsumerOption) (pubsub.Consumer, error) {
	if callback == nil {
		return nil, errors.Wrap(pubsubErrors.UndefinedEventCallbackErr, "cannot create a consumer with a 'nil' callback")
	}
	return &DefaultConsumer{
		callback: callback,
	}, nil
}

type DefaultConsumer struct {
	callback pubsub.EventCallback
}

func (c *DefaultConsumer) Consume(waitable concurrency.Waitable, event pubsub.Event) <-chan error {
	errC := make(chan error)
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
