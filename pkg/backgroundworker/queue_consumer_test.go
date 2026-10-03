package backgroundworker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueConsumer_ProcessesItems(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3

	var count atomic.Int32
	c := &QueueConsumer[int]{
		Name:   "test-process",
		Source: ch,
		Handle: func(_ context.Context, _ int) error {
			count.Add(1)
			return nil
		},
	}
	c.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	c.Stop()
	assert.Equal(t, int32(3), count.Load())
}

func TestQueueConsumer_Concurrency(t *testing.T) {
	ch := make(chan int, 10)
	for i := range 10 {
		ch <- i
	}

	var peak atomic.Int32
	var current atomic.Int32

	c := &QueueConsumer[int]{
		Name:        "test-concurrency",
		Source:      ch,
		Concurrency: 3,
		Handle: func(_ context.Context, _ int) error {
			cur := current.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			current.Add(-1)
			return nil
		},
	}
	c.Start(context.Background())
	time.Sleep(300 * time.Millisecond)
	c.Stop()

	assert.Equal(t, int32(3), peak.Load(), "peak concurrency should equal Concurrency limit")
}

func TestQueueConsumer_StopDrainsInFlight(t *testing.T) {
	ch := make(chan int, 1)

	started := make(chan struct{})
	var completed atomic.Bool

	c := &QueueConsumer[int]{
		Name:   "test-drain",
		Source: ch,
		Handle: func(_ context.Context, _ int) error {
			close(started)
			time.Sleep(100 * time.Millisecond)
			completed.Store(true)
			return nil
		},
	}
	c.Start(context.Background())
	ch <- 1

	<-started
	c.Stop()

	assert.True(t, completed.Load(), "Stop should wait for in-flight handler to complete")

	s := c.Status()
	assert.Equal(t, int64(1), s.RunCount)
	assert.Equal(t, int64(0), s.InFlight)
}

func TestQueueConsumer_ClosedChannel(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)

	var count atomic.Int32
	c := &QueueConsumer[int]{
		Name:   "test-closed",
		Source: ch,
		Handle: func(_ context.Context, _ int) error {
			count.Add(1)
			return nil
		},
	}
	c.Start(context.Background())

	// Wait for the consumer to process items and exit naturally due to closed channel.
	time.Sleep(100 * time.Millisecond)

	// Stop should return quickly since the loop already exited.
	done := make(chan struct{})
	go func() {
		c.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not exit after channel close")
	}

	assert.Equal(t, int32(2), count.Load())
	assert.Equal(t, "stopped", c.Status().State)
}

func TestQueueConsumer_StatusTracking(t *testing.T) {
	ch := make(chan int, 5)
	for i := range 5 {
		ch <- i
	}

	callNum := atomic.Int32{}
	c := &QueueConsumer[int]{
		Name:        "test-status",
		Source:      ch,
		Concurrency: 2,
		Handle: func(_ context.Context, _ int) error {
			n := callNum.Add(1)
			if n%2 == 0 {
				return fmt.Errorf("even call %d", n)
			}
			return nil
		},
	}

	// Before start.
	s := c.Status()
	assert.Equal(t, "test-status", s.Name)
	assert.Equal(t, "queue", s.Kind)
	assert.Equal(t, "idle", s.State)
	assert.Equal(t, 2, s.Concurrency)
	assert.Zero(t, s.RunCount)

	c.Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	c.Stop()

	s = c.Status()
	assert.Equal(t, "stopped", s.State)
	assert.Equal(t, int64(5), s.RunCount)
	assert.Greater(t, s.ErrorCount, int64(0))
	assert.Less(t, s.ErrorCount, s.RunCount)
	assert.False(t, s.LastRunTime.IsZero())
	assert.Greater(t, s.LastRunDur, time.Duration(0))
}

func TestQueueConsumer_StopBeforeStart(t *testing.T) {
	ch := make(chan int)
	c := &QueueConsumer[int]{
		Name:   "test-no-start",
		Source: ch,
		Handle: func(_ context.Context, _ int) error { return nil },
	}
	assert.NotPanics(t, func() { c.Stop() })
	assert.Equal(t, "idle", c.Status().State)
}

func TestQueueConsumer_DefaultConcurrency(t *testing.T) {
	ch := make(chan int, 5)
	for i := range 5 {
		ch <- i
	}

	var peak atomic.Int32
	var current atomic.Int32

	c := &QueueConsumer[int]{
		Name:   "test-default-concurrency",
		Source: ch,
		Handle: func(_ context.Context, _ int) error {
			cur := current.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			current.Add(-1)
			return nil
		},
	}
	c.Start(context.Background())
	time.Sleep(250 * time.Millisecond)
	c.Stop()
	assert.Equal(t, int32(1), peak.Load(), "default concurrency should be 1")
}

func TestQueueConsumer_Registerable(t *testing.T) {
	ch := make(chan int)
	c := &QueueConsumer[int]{
		Name:   "test-registerable",
		Source: ch,
		Handle: func(_ context.Context, _ int) error { return nil },
	}
	var _ Registerable = c

	r := NewRegistry()
	r.Register(c)
	all := r.All()
	require.Len(t, all, 1)
	assert.Equal(t, "test-registerable", all[0].Name)
	assert.Equal(t, "queue", all[0].Kind)
}

func TestQueueConsumer_InFlightTracking(t *testing.T) {
	ch := make(chan int, 3)

	var started sync.WaitGroup
	started.Add(3)
	release := make(chan struct{})

	c := &QueueConsumer[int]{
		Name:        "test-inflight",
		Source:      ch,
		Concurrency: 3,
		Handle: func(_ context.Context, _ int) error {
			started.Done()
			<-release
			return nil
		},
	}
	c.Start(context.Background())

	ch <- 1
	ch <- 2
	ch <- 3

	started.Wait()

	s := c.Status()
	assert.Equal(t, int64(3), s.InFlight)

	close(release)
	time.Sleep(50 * time.Millisecond)

	s = c.Status()
	assert.Equal(t, int64(0), s.InFlight)

	c.Stop()
}
