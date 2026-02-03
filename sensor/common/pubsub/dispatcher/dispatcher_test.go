package dispatcher

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	s.defaultLanes = []pubsub.LaneConfig{
		laneConfig,
	}
	d, lanes := newDispatcher(s.T(), s.ctrl, s.defaultLanes)
	require.Len(s.T(), lanes, 1)
	s.d = d
	s.lane = lanes[0]
}

func TestDispatcher(t *testing.T) {
	suite.Run(t, new(dispatcherSuite))
}

func (s *dispatcherSuite) Test_WithLaneConfigs() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error if no lanes are configured", func() {
		d, err := NewDispatcher()
		s.Assert().Error(err)
		s.Assert().Nil(d)
	})
	s.Run("should error with empty lane configuration", func() {
		d, err := NewDispatcher(WithLaneConfigs([]pubsub.LaneConfig{}))
		s.Assert().Error(err)
		s.Assert().Nil(d)
	})
	s.Run("should error if lanes fail to be created", func() {
		lc := mocks.NewMockLaneConfig(s.ctrl)
		lc.EXPECT().LaneID().Times(2).Return(pubsub.DefaultLane)
		lc.EXPECT().NewLane().Times(1).Return(nil)
		d, err := NewDispatcher(WithLaneConfigs([]pubsub.LaneConfig{lc}))
		s.Assert().Error(err)
		s.Assert().Nil(d)
	})
	s.Run("should error if duplicated lanes are configured", func() {
		lane := mocks.NewMockLane(s.ctrl)
		lc1 := mocks.NewMockLaneConfig(s.ctrl)
		lc1.EXPECT().LaneID().Times(2).Return(pubsub.DefaultLane)
		lc1.EXPECT().NewLane().Times(1).Return(lane)
		lane.EXPECT().Stop().Times(1)
		lc2 := mocks.NewMockLaneConfig(s.ctrl)
		lc2.EXPECT().LaneID().Times(2).Return(pubsub.DefaultLane)
		d, err := NewDispatcher(WithLaneConfigs([]pubsub.LaneConfig{lc1, lc2}))
		s.Assert().Error(err)
		s.Assert().Nil(d)
	})
}

func (s *dispatcherSuite) Test_RegisterConsumer() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should not register nil callback", func() {
		s.Assert().Error(s.d.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, nil))
	})
	s.Run("should error if lane RegisterConsumer fails", func() {
		s.lane.EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultConsumer), gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(errors.New("some error"))
		s.Assert().Error(s.d.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
	})
	s.Run("success case", func() {
		s.lane.EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultConsumer), gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(nil)
		s.Assert().NoError(s.d.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, func(_ pubsub.Event) error {
			return nil
		}))
	})
}

func (s *dispatcherSuite) Test_RegisterConsumerToLane() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should not register nil callback", func() {
		s.Assert().Error(s.d.RegisterConsumerToLane(pubsub.DefaultConsumer, pubsub.DefaultTopic, pubsub.DefaultLane, nil))
	})
	s.Run("should error if lane does not exist", func() {
		s.Assert().Error(s.d.RegisterConsumerToLane(pubsub.DefaultConsumer, pubsub.DefaultTopic, -1, func(_ pubsub.Event) error { return nil }))
	})
	s.Run("should error if lane RegisterConsumer fails", func() {
		s.lane.EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultConsumer), gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(errors.New("some error"))
		s.Assert().Error(s.d.RegisterConsumerToLane(pubsub.DefaultConsumer, pubsub.DefaultTopic, pubsub.DefaultLane, func(_ pubsub.Event) error { return nil }))
	})
	s.Run("success case", func() {
		s.lane.EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultConsumer), gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(nil)
		s.Assert().NoError(s.d.RegisterConsumerToLane(pubsub.DefaultConsumer, pubsub.DefaultTopic, pubsub.DefaultLane, func(_ pubsub.Event) error { return nil }))
	})
}

func (s *dispatcherSuite) Test_Publish() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	data := "some data"
	cases := map[string]struct {
		event           pubsub.Event
		callback        testCallback
		laneExpectCalls func(*mocks.MockLane, pubsub.EventCallback)
		expectError     func(assert.TestingT, error, ...interface{}) bool
		handleWaitGroup func(*sync.WaitGroup)
	}{
		"should error with nil event": {
			event:           nil,
			callback:        newCallback(noopAssertFn),
			laneExpectCalls: func(_ *mocks.MockLane, _ pubsub.EventCallback) {},
			expectError:     assert.Error,
			handleWaitGroup: triggerWaitGroup,
		},
		"should error with lane publish error": {
			event:    &testEvent{},
			callback: newCallback(noopAssertFn),
			laneExpectCalls: func(lane *mocks.MockLane, _ pubsub.EventCallback) {
				lane.EXPECT().Publish(gomock.Any()).Times(1).Return(errors.New("some error"))
			},
			expectError:     assert.Error,
			handleWaitGroup: triggerWaitGroup,
		},
		"should error with unknown lane": {
			event: &testEvent{
				lane: -1, // Unknown lane
			},
			callback:        newCallback(noopAssertFn),
			laneExpectCalls: func(_ *mocks.MockLane, _ pubsub.EventCallback) {},
			expectError:     assert.Error,
			handleWaitGroup: triggerWaitGroup,
		},
		"success": {
			event: &testEvent{
				data: data,
			},
			callback: newCallback(func(t *testing.T, event pubsub.Event) {
				eventImpl, ok := event.(*testEvent)
				require.True(t, ok)
				assert.Equal(t, data, eventImpl.data)
			}),
			laneExpectCalls: func(lane *mocks.MockLane, callback pubsub.EventCallback) {
				lane.EXPECT().Publish(gomock.Any()).Times(1).DoAndReturn(func(ev any) error {
					event, ok := ev.(*testEvent)
					require.True(s.T(), ok)
					return callback(event)
				})
			},
			expectError:     assert.NoError,
			handleWaitGroup: noopWaitGroup,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			wg := sync.WaitGroup{}
			wg.Add(1)
			d, lanes := newDispatcher(s.T(), s.ctrl, s.defaultLanes)
			callback := tCase.callback(s.T(), &wg)
			s.Require().Len(lanes, 1)
			lanes[0].EXPECT().RegisterConsumer(gomock.Eq(pubsub.DefaultConsumer), gomock.Eq(pubsub.DefaultTopic), gomock.Any()).Times(1).Return(nil)
			s.Assert().NoError(d.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, callback))
			tCase.laneExpectCalls(lanes[0], callback)
			err := d.Publish(tCase.event)
			tCase.expectError(s.T(), err)
			tCase.handleWaitGroup(&wg)
			wg.Wait()
			lanes[0].EXPECT().Stop().Times(1)
			d.Stop()
		})
	}
}

func newDispatcher(t *testing.T, ctrl *gomock.Controller, laneConfigs []pubsub.LaneConfig) (*dispatcher, []*mocks.MockLane) {
	var lanes []*mocks.MockLane
	for _, lc := range laneConfigs {
		lcMock, ok := lc.(*mocks.MockLaneConfig)
		require.True(t, ok)
		lane := mocks.NewMockLane(ctrl)
		lcMock.EXPECT().LaneID().Times(2).Return(pubsub.DefaultLane)
		lcMock.EXPECT().NewLane().Times(1).Return(lane)
		lanes = append(lanes, lane)
	}
	d, err := NewDispatcher(WithLaneConfigs(laneConfigs))
	require.NoError(t, err)
	return d, lanes
}

type testCallback func(*testing.T, *sync.WaitGroup) pubsub.EventCallback

func newCallback(assertFn func(*testing.T, pubsub.Event)) testCallback {
	return func(t *testing.T, wg *sync.WaitGroup) pubsub.EventCallback {
		return func(event pubsub.Event) error {
			defer wg.Done()
			assertFn(t, event)
			return nil
		}
	}
}

func noopAssertFn(_ *testing.T, _ pubsub.Event) {
}

func triggerWaitGroup(wg *sync.WaitGroup) {
	wg.Done()
}

func noopWaitGroup(_ *sync.WaitGroup) {
}

type testEvent struct {
	topic pubsub.Topic
	lane  pubsub.LaneID
	data  string
}

func (e *testEvent) Topic() pubsub.Topic {
	return e.topic
}

func (e *testEvent) Lane() pubsub.LaneID {
	return e.lane
}
