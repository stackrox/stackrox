package consumer

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
)

type Consumer struct {
	callback pubsub.EventCallback
}

func (c *Consumer) isCallbackConfigured(waitable concurrency.Waitable, errC chan<- error) bool {
	if c.callback == nil {
		go func() {
			defer close(errC)
			select {
			case errC <- pubsubErrors.UndefinedEventCallbackErr:
			case <-waitable.Done():
			}
		}()
		return false
	}
	return true
}
