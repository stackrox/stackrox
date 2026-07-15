//go:build sql_integration

package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

type ListenNotifySuite struct {
	suite.Suite
	pool postgres.DB
	ctx  context.Context
}

func TestListenNotifySuite(t *testing.T) {
	suite.Run(t, new(ListenNotifySuite))
}

func (s *ListenNotifySuite) SetupTest() {
	s.ctx = context.Background()
	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.pool = pool
}

func (s *ListenNotifySuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *ListenNotifySuite) TestNotifyAndReceive() {
	received := make(chan struct {
		channel string
		payload string
	}, 1)

	handler := func(channel, payload string) {
		received <- struct {
			channel string
			payload string
		}{channel, payload}
	}

	listener := NewListener(s.pool, handler, "test_channel")

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	go listener.Listen(ctx)

	// Give the listener time to connect and register LISTEN.
	time.Sleep(200 * time.Millisecond)

	err := Notify(s.ctx, s.pool, "test_channel", "hello")
	s.Require().NoError(err)

	select {
	case msg := <-received:
		s.Equal("test_channel", msg.channel)
		s.Equal("hello", msg.payload)
	case <-time.After(5 * time.Second):
		s.Fail("timed out waiting for notification")
	}
}

func (s *ListenNotifySuite) TestMultipleChannels() {
	received := make(chan struct {
		channel string
		payload string
	}, 10)

	handler := func(channel, payload string) {
		received <- struct {
			channel string
			payload string
		}{channel, payload}
	}

	listener := NewListener(s.pool, handler, "chan_a", "chan_b")

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	go listener.Listen(ctx)
	time.Sleep(200 * time.Millisecond)

	s.Require().NoError(Notify(s.ctx, s.pool, "chan_a", "msg_a"))
	s.Require().NoError(Notify(s.ctx, s.pool, "chan_b", "msg_b"))

	messages := make(map[string]string)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-received:
			messages[msg.channel] = msg.payload
		case <-time.After(5 * time.Second):
			s.Fail("timed out waiting for notification")
		}
	}

	s.Equal("msg_a", messages["chan_a"])
	s.Equal("msg_b", messages["chan_b"])
}

func (s *ListenNotifySuite) TestUnrelatedChannelIgnored() {
	received := make(chan string, 1)

	handler := func(channel, payload string) {
		received <- payload
	}

	listener := NewListener(s.pool, handler, "my_channel")

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	go listener.Listen(ctx)
	time.Sleep(200 * time.Millisecond)

	s.Require().NoError(Notify(s.ctx, s.pool, "other_channel", "should_not_see"))

	select {
	case <-received:
		s.Fail("should not have received notification on unrelated channel")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *ListenNotifySuite) TestContextCancellationStopsListener() {
	handler := func(channel, payload string) {}

	listener := NewListener(s.pool, handler, "stop_test")

	ctx, cancel := context.WithCancel(s.ctx)
	done := make(chan struct{})
	go func() {
		listener.Listen(ctx)
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.Fail("listener did not stop after context cancellation")
	}
}
