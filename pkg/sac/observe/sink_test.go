package observe

import (
	"context"
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestSink(t *testing.T) {
	suite.Run(t, new(sinkTestSuite))
}

type sinkTestSuite struct {
	suite.Suite
	sink AuthzTraceSink
}

func (s *sinkTestSuite) SetupTest() {
	// Create new sink without channels.
	s.sink = NewAuthzTraceSink()
}

func (s *sinkTestSuite) TestIsActive() {
	s.False(s.sink.IsActive(), "sink with no streams should not be active")

	_ = s.sink.Subscribe(context.Background())
	s.True(s.sink.IsActive(), "sink with streams should be active")
}

func (s *sinkTestSuite) TestSubscribe() {
	ctx, cancel := context.WithCancel(context.Background())
	c := s.sink.Subscribe(ctx)
	trace := &v1.AuthorizationTraceResponse{}
	go s.sink.PublishAuthzTrace(trace)
	msg := <-c
	s.Equal(trace, msg, "subscribed channel should receive the message")

	cancel()
	s.sink.PublishAuthzTrace(trace)
	s.False(s.sink.IsActive(), "cancel should result in sink unsubscribing from channel")
}

func (s *sinkTestSuite) TestPublishAuthzTrace() {
	// AuthzTraceSink with no streams does not block execution.
	s.sink.PublishAuthzTrace(&v1.AuthorizationTraceResponse{})

	// Message gets delivered to all subscribed channels.
	trace := &v1.AuthorizationTraceResponse{}
	c1 := s.sink.Subscribe(context.Background())
	c2 := s.sink.Subscribe(context.Background())
	go s.sink.PublishAuthzTrace(trace)
	// We can expect these two messages to be sent without blocking because channels have buffer size (1).
	msg1 := <-c1
	s.Equal(trace, msg1, "message should be delivered to first channel")
	msg2 := <-c2
	s.Equal(trace, msg2, "message should be delivered to second channel")
}
