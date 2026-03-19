package handler

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResourceID = "test-resource"

func TestWithRetryable(t *testing.T) {
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
		t.Run(name, func(t *testing.T) {
			counterMux := &sync.Mutex{}
			counter := 0

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			umh := NewUnconfirmedMessageHandler(ctx, "test", cc.baseDuration)
			// sending loop
			for _, tt := range cc.sendAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					t.Logf("Sending test message")
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
			assert.Equal(t, cc.expectedRetries, counter)
		})
	}
}

func TestMultipleResources(t *testing.T) {
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
		assert.Equal(t, "resource-2", resourceID, "should retry resource-2")
	case <-time.After(baseInterval + 500*time.Millisecond):
		t.Fatal("expected retry for resource-2")
	}
}

func TestOnACKCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	baseInterval := 50 * time.Millisecond
	umh := NewUnconfirmedMessageHandler(ctx, "test", baseInterval)

	var ackedResources []string
	var ackMu sync.Mutex
	wg := concurrency.NewWaitGroup(3)
	umh.OnACK(func(resourceID string) {
		defer wg.Add(-1)
		ackMu.Lock()
		defer ackMu.Unlock()
		ackedResources = append(ackedResources, resourceID)
	})

	// ACK should invoke the callback
	umh.HandleACK("resource-ack-1")
	umh.HandleACK("resource-ack-2")
	umh.HandleACK("resource-ack-3")

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected all callbacks to be invoked")
	case <-wg.Done():
	}

	ackMu.Lock()
	defer ackMu.Unlock()
	assert.Len(t, ackedResources, 3)
	assert.ElementsMatch(t, []string{"resource-ack-1", "resource-ack-2", "resource-ack-3"}, ackedResources)
}

func TestShutdown(t *testing.T) {
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
		t.Fatal("Cleanup should complete within timeout")
	}

	// After shutdown, retryCommand channel should be closed (receive returns zero value, ok=false)
	select {
	case rid, ok := <-umh.RetryCommand():
		if ok {
			t.Fatalf("expected channel to be closed or empty, got value: rid=%s", rid)
		}
		// ok=false means channel is closed, which is expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Channel should be closed and not block")
	}
}

func TestOperationsOnDeadHandler(t *testing.T) {
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
		t.Fatal("Cleanup should complete within timeout")
	}

	// All operations on dead handler should be safe (no panic, no race, no blocking)
	require.NotPanics(t, func() {
		// These should not panic on a dead handler
		umh.ObserveSending("resource-1")
		umh.ObserveSending("resource-2")
		umh.HandleACK("resource-1")
		umh.HandleACK("unknown-resource")
		umh.HandleNACK("resource-2")
		umh.HandleNACK("unknown-resource")

		// Channel accessor should return closed channel
		_ = umh.RetryCommand()
		_ = umh.Stopped()
	})

	// Verify RetryCommand channel is closed (receive should not block)
	select {
	case _, ok := <-umh.RetryCommand():
		assert.False(t, ok, "RetryCommand channel should be closed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RetryCommand channel should not block")
	}

	// Stopped should already be signaled
	assert.True(t, umh.Stopped().IsDone(), "Stopped should be signaled")
}

func TestOnTimerFiredDoesNotBlockWhenRetryQueueFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		umh := NewUnconfirmedMessageHandler(ctx, "test-full-queue", 100*time.Millisecond)

		// Seed one resource with an unacked sending so timer logic is exercised.
		umh.mu.Lock()
		umh.resources["resource-full-queue"] = &resourceState{
			numUnackedSendings: 1,
		}
		umh.mu.Unlock()

		// Fill the single-slot retry queue to force coalescing/default branch.
		umh.retryCommandCh <- "already-queued"

		done := concurrency.NewSignal()
		go func() {
			umh.onTimerFired("resource-full-queue")
			done.Signal()
		}()

		// Wait until all runnable goroutines settle before asserting completion.
		synctest.Wait()
		select {
		case <-done.Done():
		default:
			t.Fatal("onTimerFired should not block when retry queue is full")
		}

		// Cleanup should complete without relying on real-time deadlines.
		cancel()
		synctest.Wait()
		select {
		case <-umh.Stopped().Done():
		default:
			t.Fatal("cleanup should complete")
		}
	})
}

func TestShutdownWithPendingBufferedRetrySignal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		umh := NewUnconfirmedMessageHandler(ctx, "test-shutdown-buffered", time.Second)

		// Put one pending retry signal in the buffered channel without a consumer.
		umh.retryCommandCh <- "pending-retry"
		cancel()
		synctest.Wait()

		select {
		case <-umh.Stopped().Done():
		default:
			t.Fatal("cleanup should complete")
		}

		// After shutdown, a pending buffered value may be drained first, but then the channel must be closed.
		ch := umh.RetryCommand()
		select {
		case _, ok := <-ch:
			// ok=true means we drained the buffered value; ok=false means channel was already closed.
			if ok {
				select {
				case _, ok := <-ch:
					assert.False(t, ok, "retry channel should be closed after draining pending buffered value")
				default:
					t.Fatal("retry channel close check should not block")
				}
			}
		default:
			t.Fatal("retry channel drain/close should not block")
		}
	})
}
