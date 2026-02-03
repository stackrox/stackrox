package lane

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	log = logging.LoggerForModule()
)

type Config[T pubsub.Lane] struct {
	id          pubsub.LaneID
	opts        []pubsub.LaneOption[T]
	newConsumer pubsub.NewConsumer
}

func (c *Config[T]) LaneID() pubsub.LaneID {
	return c.id
}

type Lane struct {
	id            pubsub.LaneID
	newConsumerFn pubsub.NewConsumer
	consumerLock  sync.RWMutex
	consumers     map[pubsub.Topic][]pubsub.Consumer
	consumerOpts  []pubsub.ConsumerOption
}

func (l *Lane) Stop() {
	concurrency.WithLock(&l.consumerLock, func() {
		for _, consumersPerTopic := range l.consumers {
			for _, c := range consumersPerTopic {
				c.Stop()
			}
		}
	})
}
