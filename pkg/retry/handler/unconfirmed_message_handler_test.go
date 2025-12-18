package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
)

const testResourceID = "test-resource"

func TestUnconfirmedMessageHandler(t *testing.T) {
	suite.Run(t, new(UnconfirmedMessageHandlerTestSuite))
}

type UnconfirmedMessageHandlerTestSuite struct {
	suite.Suite
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestWithRetryable() {
	cases := map[string]struct {
		baseDuration    time.Duration
		wait            time.Duration
		expectedRetries int
		sendAfter       []time.Duration
		ackAfter        []time.Duration
		nackAfter       []time.Duration
	}{
		"should retry once when 0 acks": {
			baseDuration:    time.Second,
			wait:            1100 * time.Millisecond,
			expectedRetries: 1,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should not retry when acks arrives immediately": {
			baseDuration:    time.Second,
			wait:            500 * time.Millisecond,
			expectedRetries: 0,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{10 * time.Millisecond},
			nackAfter:       []time.Duration{},
		},
		"should retry 3 times within 6 seconds when base set to 1s": {
			baseDuration:    time.Second,
			wait:            6100 * time.Millisecond, // Retries after: 1s, 2s, 3s
			expectedRetries: 3,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should not reset retries if the first messsage is unacked and second message is sent": {
			baseDuration: time.Second,
			wait:         9100 * time.Millisecond,
			// Withouth reset, it retries 3 times in 9s: (send) 1s, 2s, (send), 3s, (test stop) 4s, 5s.
			// With reset, it would retry 4 times in 9s: (send) 1s, 2s, (send), 1s, 2s, 3s, (test stop).
			expectedRetries: 3,
			sendAfter:       []time.Duration{1 * time.Millisecond, 4100 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should retry normally when nack is received": {
			baseDuration:    time.Second,
			wait:            1100 * time.Millisecond,
			expectedRetries: 1,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{3 * time.Millisecond},
		},
	}

	for name, cc := range cases {
		suite.Run(name, func() {
			counterMux := &sync.Mutex{}
			counter := 0

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			umh := NewUnconfirmedMessageHandler(ctx, "test", cc.baseDuration)
			// sending loop
			for _, tt := range cc.sendAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					suite.T().Logf("Sending test message")
					umh.ObserveSending(testResourceID)
				}(tt)
			}
			// acking loop
			for _, tt := range cc.ackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					umh.HandleACK(testResourceID)
				}(tt)
			}
			// nacking loop
			for _, tt := range cc.nackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					umh.HandleNACK(testResourceID)
				}(tt)
			}
			// retry-counting loop
			go func() {
				for range umh.RetryCommand() {
					concurrency.WithLock(counterMux, func() {
						counter++
					})
				}
			}()
			<-time.After(cc.wait)

			counterMux.Lock()
			defer counterMux.Unlock()
			suite.Equal(cc.expectedRetries, counter)
		})
	}
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestMultipleResources() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseInterval := time.Second
	umh := NewUnconfirmedMessageHandler(ctx, "test", baseInterval)

	// Send for two different resources
	umh.ObserveSending("resource-1")
	umh.ObserveSending("resource-2")

	// ACK only resource-1
	umh.HandleACK("resource-1")

	// Wait for resource-2 to trigger retry (baseInterval + margin)
	select {
	case resourceID := <-umh.RetryCommand():
		suite.Equal("resource-2", resourceID, "should retry resource-2")
	case <-time.After(baseInterval + 500*time.Millisecond):
		suite.Fail("expected retry for resource-2")
	}
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestAckedResources() {
	ctx, cancel := context.WithCancel(context.Background())
	baseInterval := 50 * time.Millisecond
	umh := NewUnconfirmedMessageHandler(ctx, "test", baseInterval)

	// ACK should emit resource ID on AckedResources.
	umh.HandleACK("resource-ack-1")
	umh.HandleACK("resource-ack-2")
	umh.HandleACK("resource-ack-3")

	select {
	case got, ok := <-umh.AckedResources():
		suite.True(ok)
		suite.Equal("resource-ack-1", got)
	case <-time.After(time.Second):
		suite.Fail("expected resource-ack-<n> to be acked")
	}
	cancel()
	// Acks channel is still open and not drained
	got, ok := <-umh.AckedResources()
	suite.True(ok)
	suite.Equal("resource-ack-2", got)

}

func (suite *UnconfirmedMessageHandlerTestSuite) TestShutdown() {
	ctx, cancel := context.WithCancel(context.Background())
	baseInterval := 50 * time.Millisecond
	umh := NewUnconfirmedMessageHandler(ctx, "test", baseInterval)
	// Observe sending, ACK, and trigger shutdown
	umh.ObserveSending("resource-shutdown")
	umh.HandleACK("resource-shutdown")
	cancel()

	// Wait for cleanup to complete using the stopper
	select {
	case <-umh.Stopped().Done():
		// Cleanup complete
	case <-time.After(time.Second):
		suite.Fail("Cleanup should complete within timeout")
	}

	// After shutdown, retryCommand channel should be closed (receive returns zero value, ok=false)
	select {
	case rid, ok := <-umh.RetryCommand():
		if ok {
			suite.Failf("Expected channel to be closed or empty, got value", "rid=%s", rid)
		}
		// ok=false means channel is closed, which is expected
	case <-time.After(100 * time.Millisecond):
		suite.Fail("Channel should be closed and not block")
	}
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestOperationsOnDeadHandler() {
	ctx, cancel := context.WithCancel(context.Background())
	baseInterval := 50 * time.Millisecond
	umh := NewUnconfirmedMessageHandler(ctx, "test-dead", baseInterval)

	// Cancel immediately to shut down the handler
	cancel()

	// Wait for cleanup to complete
	select {
	case <-umh.Stopped().Done():
		// Cleanup complete
	case <-time.After(time.Second):
		suite.Fail("Cleanup should complete within timeout")
	}

	// All operations on dead handler should be safe (no panic, no race, no blocking)
	suite.NotPanics(func() {
		// These should not panic on a dead handler
		umh.ObserveSending("resource-1")
		umh.ObserveSending("resource-2")
		umh.HandleACK("resource-1")
		umh.HandleACK("unknown-resource")
		umh.HandleNACK("resource-2")
		umh.HandleNACK("unknown-resource")

		// Channel accessors should return closed channels
		_ = umh.RetryCommand()
		_ = umh.AckedResources()
		_ = umh.Stopped()
	})

	// Verify channels are closed (receive should not block)
	select {
	case _, ok := <-umh.RetryCommand():
		suite.False(ok, "RetryCommand channel should be closed")
	case <-time.After(100 * time.Millisecond):
		suite.Fail("RetryCommand channel should not block")
	}

	select {
	case _, ok := <-umh.AckedResources():
		suite.False(ok, "AckedResources channel should be closed")
	case <-time.After(100 * time.Millisecond):
		suite.Fail("AckedResources channel should not block")
	}

	// Stopped should already be signaled
	suite.True(umh.Stopped().IsDone(), "Stopped should be signaled")
}
