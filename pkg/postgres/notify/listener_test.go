//go:build sql_integration

package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestListenerLifecycle(t *testing.T) {
	db, err := postgres.Connect(context.Background(), pgtest.GetConnectionString(t))
	require.NoError(t, err)
	t.Cleanup(db.Close)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ready := make(chan struct{}, 1)
	received := make(chan string, 1)
	disconnected := make(chan error, 1)
	done := make(chan struct{})
	l := NewListenerWithHooks(db, func(_, payload string) { received <- payload }, LifecycleHooks{
		AfterListen: func(ctx context.Context) error {
			// This write uses another connection. Receiving it proves LISTEN
			// committed before the synchronization hook ran.
			if err := Notify(ctx, db, "lifecycle", "after listen"); err != nil {
				return err
			}
			ready <- struct{}{}
			return nil
		},
		OnDisconnect: func(err error) { disconnected <- err },
	}, "lifecycle")
	go func() { l.Listen(ctx); close(done) }()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("LISTEN was not ready")
	}
	select {
	case payload := <-received:
		require.Equal(t, "after listen", payload)
	case <-time.After(10 * time.Second):
		t.Fatal("notification from synchronization hook was lost")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("listener did not close")
	}
	require.ErrorIs(t, <-disconnected, context.Canceled)
}

func TestListenerSynchronizationFailure(t *testing.T) {
	db, err := postgres.Connect(context.Background(), pgtest.GetConnectionString(t))
	require.NoError(t, err)
	t.Cleanup(db.Close)
	failure := errors.New("snapshot failed")
	l := NewListenerWithHooks(db, func(_, _ string) { t.Error("dispatched after failed synchronization") },
		LifecycleHooks{AfterListen: func(context.Context) error { return failure }}, "failure")
	require.ErrorIs(t, l.listenLoop(context.Background()), failure)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var disconnected error
	l.hooks.OnDisconnect = func(err error) { disconnected = err; cancel() }
	l.Listen(ctx)
	require.ErrorIs(t, disconnected, failure)
}

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

func (s *ListenNotifySuite) startListener(ctx context.Context, handler Handler, channels ...string) <-chan struct{} {
	ready, done := make(chan struct{}), make(chan struct{})
	listener := NewListenerWithHooks(s.pool, handler, LifecycleHooks{
		AfterListen: func(context.Context) error { close(ready); return nil },
	}, channels...)
	go func() { listener.Listen(ctx); close(done) }()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		s.T().Fatal("listener was not ready")
	}
	return done
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

	ctx, cancel := context.WithCancel(s.ctx)
	done := s.startListener(ctx, handler, "test_channel")
	defer func() { cancel(); <-done }()

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

	ctx, cancel := context.WithCancel(s.ctx)
	done := s.startListener(ctx, handler, "chan_a", "chan_b")
	defer func() { cancel(); <-done }()

	s.Require().NoError(Notify(s.ctx, s.pool, "chan_a", "msg_a"))
	s.Require().NoError(Notify(s.ctx, s.pool, "chan_b", "msg_b"))

	messages := make(map[string]string)
	for range 2 {
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

	ctx, cancel := context.WithCancel(s.ctx)
	done := s.startListener(ctx, handler, "my_channel")
	defer func() { cancel(); <-done }()

	s.Require().NoError(Notify(s.ctx, s.pool, "other_channel", "should_not_see"))
	s.Require().NoError(Notify(s.ctx, s.pool, "my_channel", "barrier"))

	select {
	case payload := <-received:
		s.Equal("barrier", payload)
	case <-time.After(5 * time.Second):
		s.Fail("did not receive barrier")
	}
}

func (s *ListenNotifySuite) TestContextCancellationStopsListener() {
	handler := func(channel, payload string) {}

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	done := s.startListener(ctx, handler, "stop_test")
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.Fail("listener did not stop after context cancellation")
	}
}
