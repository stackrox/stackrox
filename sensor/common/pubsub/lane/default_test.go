package lane

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

type defaultLaneSuite struct {
	suite.Suite
}

func TestDefaultLane(t *testing.T) {
	suite.Run(t, new(defaultLaneSuite))
}

func (s *defaultLaneSuite) TestNewLaneOptions() {
	defer goleak.VerifyNone(s.T())
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
}

func (s *defaultLaneSuite) TestPublish() {
	defer goleak.VerifyNone(s.T())
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
		time.Sleep(500 * time.Millisecond)
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
