package dispatcher

import (
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
)

type dispatcherSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	defaultLanes []pubsub.LaneConfig
	d            *dispatcher
	lane         *mocks.MockLane
}

func (s *dispatcherSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	laneConfig := mocks.NewMockLaneConfig(s.ctrl)
	s.lane = mocks.NewMockLane(s.ctrl)
	s.defaultLanes = []pubsub.LaneConfig{
		laneConfig,
	}
	laneConfig.EXPECT().LaneID().Times(2).Return(pubsub.DefaultLane)
	laneConfig.EXPECT().NewLane().Times(1).Return(s.lane)
	ps, err := NewDispatcher(WithLaneConfigs(s.defaultLanes))
	require.NoError(s.T(), err)
	s.d = ps
}

func TestDispatcher(t *testing.T) {
	suite.Run(t, new(dispatcherSuite))
}

func (s *dispatcherSuite) Test_Dispatcher() {
	defer goleak.VerifyNone(s.T())
	wg := sync.WaitGroup{}
	wg.Add(1)
	data := "some data"
	var callback pubsub.EventCallback = func(event pubsub.Event) error {
		defer wg.Done()
		eventImpl, ok := event.(*testEvent)
		require.True(s.T(), ok)
		assert.Equal(s.T(), data, eventImpl.data)
		return nil
	}
	s.lane.EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(nil)
	assert.NoError(s.T(), s.d.RegisterConsumer(pubsub.DefaultTopic, callback))
	ev := &testEvent{
		data: data,
	}
	s.lane.EXPECT().Publish(gomock.Eq(ev)).DoAndReturn(func(ev any) error {
		event, ok := ev.(*testEvent)
		require.True(s.T(), ok)
		return callback(event)
	})
	assert.NoError(s.T(), s.d.Publish(ev))
	wg.Wait()
	s.lane.EXPECT().Stop().Times(1)
	s.d.Stop()
}

type testEvent struct {
	data string
}

func (e *testEvent) Topic() pubsub.Topic {
	return pubsub.DefaultTopic
}

func (e *testEvent) Lane() pubsub.LaneID {
	return pubsub.DefaultLane
}
