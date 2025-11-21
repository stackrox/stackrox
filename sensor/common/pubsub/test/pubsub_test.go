package test

import (
	"sync"
	"testing"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

type pubsubSuite struct {
	suite.Suite
}

func TestPubSubSystem(t *testing.T) {
	suite.Run(t, new(pubsubSuite))
}

type testEvent struct {
	data string
}

func (t *testEvent) Topic() pubsub.Topic {
	return pubsub.DefaultTopic
}

func (t *testEvent) Lane() pubsub.LaneID {
	return pubsub.DefaultLane
}

func (s *pubsubSuite) Test_PubSub() {
	defer goleak.VerifyNone(s.T())
	ps, err := dispatcher.NewDispatcher(dispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewDefaultLane(pubsub.DefaultLane),
	}))
	assert.NoError(s.T(), err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	data := "some data"
	callback := func(event pubsub.Event) error {
		defer wg.Done()
		eventImpl, ok := event.(*testEvent)
		require.True(s.T(), ok)
		assert.Equal(s.T(), data, eventImpl.data)
		return nil
	}
	assert.NoError(s.T(), ps.RegisterConsumer(pubsub.DefaultTopic, callback))
	assert.NoError(s.T(), ps.Publish(&testEvent{data: data}))
	wg.Wait()
	ps.Stop()
}
