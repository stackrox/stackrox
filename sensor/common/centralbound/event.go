package centralbound

import (
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// CentralBoundEvent wraps a message destined for Central as a PubSub event.
// Components publish these instead of writing directly to their ResponsesC channel.
type CentralBoundEvent struct {
	Msg *message.ExpiringMessage
}

func (e *CentralBoundEvent) Topic() pubsub.Topic { return pubsub.CentralBoundTopic }
func (e *CentralBoundEvent) Lane() pubsub.LaneID { return pubsub.CentralBoundLane }

// IsExpired delegates to the wrapped message's context.
func (e *CentralBoundEvent) IsExpired() bool {
	if e.Msg == nil {
		return false
	}
	return e.Msg.IsExpired()
}
