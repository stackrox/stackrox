package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stretchr/testify/suite"
)

type defaultConsumerSuite struct {
	suite.Suite
}

func TestDefaultConsumer(t *testing.T) {
	suite.Run(t, new(defaultConsumerSuite))
}

func (s *defaultConsumerSuite) TestConsume() {
	defer goleak.AssertNoGoroutineLeaks(s.T())
	s.Run("should error with nil callback", func() {
		c, err := NewDefaultConsumer(nil)
		s.Assert().Error(err)
		s.Assert().Nil(c)
	})
	s.Run("should unblock if waitable is done", func() {
		callbackDone := concurrency.NewSignal()
		c, err := NewDefaultConsumer(func(_ pubsub.Event) error {
			defer callbackDone.Signal()
			return errors.New("some error")
		})
		s.Assert().NoError(err)
		ctx, cancel := context.WithCancel(context.Background())
		_ = c.Consume(ctx, &testEvent{})
		select {
		case <-callbackDone.Done():
		case <-time.After(500 * time.Millisecond):
			s.FailNow("callback should be done")
		}
		cancel()
		// The test will fail if there are goroutine leaks
	})
	s.Run("consume event error", func() {
		data := "some data"
		consumerSignal := concurrency.NewSignal()
		c, err := NewDefaultConsumer(func(event pubsub.Event) error {
			defer consumerSignal.Signal()
			eventImpl, ok := event.(*testEvent)
			s.Require().True(ok)
			s.Assert().Equal(data, eventImpl.data)
			return errors.New("some error")
		})
		s.Assert().NoError(err)
		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{
			data: data,
		})
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for the event to be consumed")
		case <-consumerSignal.Done():
		}
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for error")
		case err, ok := <-errC:
			s.Assert().True(ok)
			s.Assert().Error(err)
		}
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for errC to be closed")
		case err, ok := <-errC:
			s.Assert().False(ok)
			s.Assert().Nil(err)
		}
	})
	s.Run("consume event no error", func() {
		data := "some data"
		consumerSignal := concurrency.NewSignal()
		c, err := NewDefaultConsumer(func(event pubsub.Event) error {
			defer consumerSignal.Signal()
			eventImpl, ok := event.(*testEvent)
			s.Require().True(ok)
			s.Assert().Equal(data, eventImpl.data)
			return nil
		})
		s.Assert().NoError(err)
		ctx := context.Background()
		errC := c.Consume(ctx, &testEvent{
			data: data,
		})
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for the event to be consumed")
		case <-consumerSignal.Done():
		}
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for nil error")
		case err, ok := <-errC:
			s.Assert().True(ok)
			s.Assert().Nil(err)
		}
		select {
		case <-time.After(100 * time.Millisecond):
			s.FailNow("timeout waiting for errC to be closed")
		case err, ok := <-errC:
			s.Assert().False(ok)
			s.Assert().Nil(err)
		}
	})
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
