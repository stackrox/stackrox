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

type blockingLaneSuite struct {
	suite.Suite
}

func TestBlockingLane(t *testing.T) {
	suite.Run(t, new(blockingLaneSuite))
}

func (s *blockingLaneSuite) TestNewLaneOptions() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("with default options", func() {
		config := NewBlockingLane(pubsub.DefaultLane)
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*BlockingLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, laneImpl.ch.Cap())
	})
	s.Run("with default lane size", func() {
		laneSize := 10
		config := NewBlockingLane(pubsub.DefaultLane, WithBlockingLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*BlockingLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), laneSize, laneImpl.ch.Cap())
	})
	s.Run("with negative lane size", func() {
		laneSize := -1
		config := NewBlockingLane(pubsub.DefaultLane, WithBlockingLaneSize(laneSize))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*BlockingLane)
		require.True(s.T(), ok)
		assert.Equal(s.T(), 0, laneImpl.ch.Cap())
	})
	s.Run("with custom consumer", func() {
		config := NewBlockingLane(pubsub.DefaultLane, WithBlockingLaneConsumer(newTestConsumer))
		assert.Equal(s.T(), pubsub.DefaultLane, config.LaneID())
		lane := config.NewLane()
		assert.NotNil(s.T(), lane)
		defer lane.Stop()
		laneImpl, ok := lane.(*BlockingLane)
		require.True(s.T(), ok)
		assert.NotNil(s.T(), laneImpl.newConsumerFn)
	})
}

func (s *blockingLaneSuite) TestOptionPanic() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("panic if a nil NewConsumer is passed to WithBlockingLaneConsumer", func() {
		config := NewBlockingLane(pubsub.DefaultLane, WithBlockingLaneConsumer(nil))
		s.Assert().Panics(func() {
			config.NewLane()
		})
	})
}

func (s *blockingLaneSuite) TestRegisterConsumer() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error on nil callback", func() {
		lane := NewBlockingLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.Error(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, nil))
		lane.Stop()
	})
}

func (s *blockingLaneSuite) TestPublish() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("publish with blocking consumer should block", func() {
		lane := NewBlockingLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		unblockSig := concurrency.NewSignal()
		wg := sync.WaitGroup{}
		wg.Add(1)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, blockingCallback(&wg, &unblockSig)))
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
		lane := NewBlockingLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		assert.NoError(s.T(), lane.Publish(&testEvent{}))
		assert.NoError(s.T(), lane.Publish(&testEvent{}))
		lane.Stop()
	})
	s.Run("publish and consume", func() {
		lane := NewBlockingLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		data := "some data"
		consumeSignal := concurrency.NewSignal()
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic,
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
		lane := NewBlockingLane(pubsub.DefaultLane).NewLane()
		assert.NotNil(s.T(), lane)
		unblockSig := concurrency.NewSignal()
		wg := sync.WaitGroup{}
		wg.Add(1)
		assert.NoError(s.T(), lane.RegisterConsumer(pubsub.DefaultConsumer, pubsub.DefaultTopic, blockingCallback(&wg, &unblockSig)))
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

func newTestConsumer(_ pubsub.LaneID, _ pubsub.Topic, _ pubsub.ConsumerID, _ pubsub.EventCallback) (pubsub.Consumer, error) {
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
