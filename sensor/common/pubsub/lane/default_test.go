package lane

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type defaultLaneSuite struct {
	suite.Suite
}

func TestDefaultLane(t *testing.T) {
	suite.Run(t, new(defaultLaneSuite))
}

func (s *defaultLaneSuite) TestNewLaneOptions() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("with default options", func() {
		config := NewDefaultLane(pubsub.DefaultLane)
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*defaultLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, cap(laneImpl.ch))
	})
	s.Run("with default lane size", func() {
		laneSize := 10
		config := NewDefaultLane(pubsub.DefaultLane, WithDefaultLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*defaultLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), laneSize, cap(laneImpl.ch))
	})
	s.Run("with negative lane size", func() {
		laneSize := -1
		config := NewDefaultLane(pubsub.DefaultLane, WithDefaultLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*defaultLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, cap(laneImpl.ch))
	})
	s.Run("with custom consumer", func() {
		config := NewDefaultLane(pubsub.DefaultLane, WithDefaultLaneConsumer(newTestConsumer))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*defaultLane)
		require.True(s.T(), ok)
		assert.NotNil(s.T(), laneImpl.newConsumerFn)
		assert.Len(s.T(), laneImpl.consumerOpts, 0)
	})
	s.Run("with custom consumer and consumer options", func() {
		config := NewDefaultLane(pubsub.DefaultLane, WithDefaultLaneConsumer(newTestConsumer, func(_ pubsub.Consumer) {}))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*defaultLane)
		require.True(s.T(), ok)
		assert.NotNil(s.T(), laneImpl.newConsumerFn)
		assert.Len(s.T(), laneImpl.consumerOpts, 1)
	})
}

type testLaneConfig struct {
	opts []pubsub.LaneOption
}

type testLane struct{}

func (t *testLane) Publish(_ pubsub.Event) error {
	return nil
}

func (t *testLane) RegisterConsumer(_ pubsub.Topic, _ pubsub.EventCallback) error {
	return nil
}

func (t *testLane) Stop() {
}

func (lc *testLaneConfig) NewLane() pubsub.Lane {
	ret := &testLane{}
	for _, opt := range lc.opts {
		opt(ret)
	}
	return ret
}
func (lc *testLaneConfig) LaneID() pubsub.LaneID {
	return pubsub.DefaultLane
}

func (s *defaultLaneSuite) TestOptionPanic() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("panic if WithDefaultLaneSize is used in a different lane", func() {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithDefaultLaneSize(10),
			},
		}
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
	s.Run("panic if WithDefaultLaneConsumer is used in a different lane", func() {
		config := &testLaneConfig{
			opts: []pubsub.LaneOption{
				WithDefaultLaneConsumer(nil),
			},
		}
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
	s.Run("panic if a nil NewConsumer is passed to WithDefaultLaneConsumer", func() {
		config := NewDefaultLane(pubsub.DefaultLane, WithDefaultLaneConsumer(nil))
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
}

func (s *defaultLaneSuite) TestRegisterConsumer() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error on nil callback", func() {
		lane := NewDefaultLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.Error(s.T(), lane.RegisterConsumer(pubsub.DefaultTopic, nil))
		lane.Stop()
	})
}

func (s *defaultLaneSuite) TestPublish() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("publish with blocking consumer should block", func() {
		lane := NewDefaultLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		unblockSig := concurrency.NewSignal()
		wg := sync.WaitGroup{}
		wg.Add(1)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultTopic, blockingCallback(&wg, &unblockSig)))
		publishDone := concurrency.NewSignal()
		go func() {
			defer publishDone.Signal()
			assert.NoError(s.T(), lane.Publish(&testEvent{}))
			assert.Error(s.T(), lane.Publish(&testEvent{}))
		}()
		select {
		case <-time.After(100 * time.Millisecond):
		case <-publishDone.Done():
			s.FailNow("Publish should block if no consumers are configured")
		}
		lane.Stop()
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Publish should unblock after Stop")
		case <-publishDone.Done():
		}
		unblockSig.Signal()
		wg.Wait()
	})
	s.Run("publish with no consumer should not block", func() {
		lane := NewDefaultLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.NoError(s.T(), lane.Publish(&testEvent{}))
		assert.NoError(s.T(), lane.Publish(&testEvent{}))
		lane.Stop()
	})
	s.Run("publish and consume", func() {
		lane := NewDefaultLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		data := "some data"
		consumeSignal := concurrency.NewSignal()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultTopic,
			assertInCallback(s.T(), func(t *testing.T, event pubsub.Event) error {
				defer consumeSignal.Signal()
				eventImpl, ok := event.(*testEvent)
				require.True(t, ok)
				assert.Equal(t, data, eventImpl.data)
				return nil
			})))
		assert.NoError(s.T(), lane.Publish(&testEvent{data: data}))
		<-consumeSignal.Done()
		lane.Stop()
	})
	s.Run("stop should unblock publish", func() {
		lane := NewDefaultLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		unblockSig := concurrency.NewSignal()
		wg := sync.WaitGroup{}
		wg.Add(1)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultTopic, blockingCallback(&wg, &unblockSig)))
		publishDone := concurrency.NewSignal()
		firstPublishCallDone := concurrency.NewSignal()
		go func() {
			defer publishDone.Signal()
			assert.NoError(s.T(), lane.Publish(&testEvent{}))
			firstPublishCallDone.Signal()
			assert.Error(s.T(), lane.Publish(&testEvent{}))
		}()
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("The fist call to publish should not block")
		case <-firstPublishCallDone.Done():
		}
		lane.Stop()
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("Publish should unblock after Stop")
		case <-publishDone.Done():
		}
		unblockSig.Signal()
		wg.Wait()
	})
}

func blockingCallback(wg *sync.WaitGroup, signal *concurrency.Signal) pubsub.EventCallback {
	return func(_ pubsub.Event) error {
		defer wg.Done()
		<-signal.Done()
		return nil
	}
}

func assertInCallback(t *testing.T, assertion func(*testing.T, pubsub.Event) error) pubsub.EventCallback {
	return func(event pubsub.Event) error {
		return assertion(t, event)
	}
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

func newTestConsumer(_ pubsub.EventCallback, _ ...pubsub.ConsumerOption) (pubsub.Consumer, error) {
	return &testCustomConsumer{}, nil
}

type testCustomConsumer struct {
}

func (c *testCustomConsumer) Consume(_ concurrency.Waitable, _ pubsub.Event) <-chan error {
	errC := make(chan error)
	defer close(errC)
	return errC
}

func (c *testCustomConsumer) Stop() {}
