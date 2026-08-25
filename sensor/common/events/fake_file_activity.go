package events

import (
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// FakeFileActivityEvent wraps a file activity message from fake workloads.
// This allows fake workloads to publish file activities through the pubsub system rather
// than writing directly to a shared channel.
type FakeFileActivityEvent struct {
	Activity *sensorAPI.FileActivity
}

func (e *FakeFileActivityEvent) Topic() pubsub.Topic {
	return pubsub.FakeFileActivityTopic
}

func (e *FakeFileActivityEvent) Lane() pubsub.LaneID {
	return pubsub.FakeFileActivityLane
}
