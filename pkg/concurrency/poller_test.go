package concurrency

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoller(t *testing.T) {
	a := assert.New(t)

	const numPollsBeforeTrue int64 = 3
	var conditionCounter int64
	condition := func() bool {
		return atomic.AddInt64(&conditionCounter, 1) >= numPollsBeforeTrue
	}
	duration := 100 * time.Millisecond

	p := NewPoller(condition, duration)
	defer func() {
		a.False(p.Stop())
	}()
	a.False(p.IsDone())
	a.True(atomic.LoadInt64(&conditionCounter) < 3)
	p.Wait()
	a.True(p.IsDone())
	a.Equal(numPollsBeforeTrue, atomic.LoadInt64(&conditionCounter))

	// Make sure there are no unnecessary polls.
	time.Sleep(2 * duration)
	a.Equal(numPollsBeforeTrue, atomic.LoadInt64(&conditionCounter))
}

func TestPollerWaitWithTimeout(t *testing.T) {
	a := assert.New(t)

	p := NewPoller(func() bool {
		return false
	}, 10*time.Millisecond)
	defer func() {
		a.True(p.Stop())
	}()

	a.False(WaitWithTimeout(p, 200*time.Millisecond))
}

func TestPollWithTimeout(t *testing.T) {
	assert.False(t, PollWithTimeout(func() bool {
		return false
	}, 5*time.Millisecond, 50*time.Millisecond))
	var ctr int32
	assert.True(t, PollWithTimeout(func() bool {
		return atomic.AddInt32(&ctr, 1) > 2
	}, 5*time.Millisecond, 50*time.Millisecond))
	assert.Equal(t, int32(3), atomic.LoadInt32(&ctr))
}

func TestPollerStops(t *testing.T) {
	a := assert.New(t)

	calledSig := NewSignal()
	var count int64
	p := NewPoller(func() bool {
		calledSig.Signal()
		atomic.AddInt64(&count, 1)
		return false
	}, 10*time.Millisecond)

	Do(&calledSig, func() {
		a.True(p.Stop())
	})

	// Make sure it stops polling beyond the first time.
	time.Sleep(100 * time.Millisecond)

	a.Equal(int64(1), atomic.LoadInt64(&count))
}
