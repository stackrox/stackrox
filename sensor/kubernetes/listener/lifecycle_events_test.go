package listener

import (
	"testing"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
)

func TestSoftRestartEvent_TopicAndLane(t *testing.T) {
	e := &SoftRestartEvent{}
	assert.Equal(t, pubsub.SoftRestartTopic, e.Topic())
	assert.Equal(t, pubsub.SoftRestartLane, e.Lane())
}

func TestResourceSyncFinishedEvent_TopicAndLane(t *testing.T) {
	e := &ResourceSyncFinishedEvent{}
	assert.Equal(t, pubsub.ResourceSyncFinishedTopic, e.Topic())
	assert.Equal(t, pubsub.ResourceSyncFinishedLane, e.Lane())
}
