package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoller(t *testing.T) {
	a := assert.New(t)

	const numPollsBeforeTrue = 3
	conditionCounter := 0
	condition := func() bool {
		conditionCounter++
		return conditionCounter >= numPollsBeforeTrue
	}
	duration := 100 * time.Millisecond

	p := NewPoller(condition, duration)
	defer func() {
		a.False(p.Stop())
	}()
	a.False(p.IsDone())
	a.True(conditionCounter < 3)
	p.Wait()
	a.True(p.IsDone())
	a.Equal(numPollsBeforeTrue, conditionCounter)

	// Make sure there are no unnecessary polls.
	time.Sleep(2 * duration)
	a.Equal(numPollsBeforeTrue, conditionCounter)
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

func TestPollerStops(t *testing.T) {
	a := assert.New(t)

	calledSig := NewSignal()
	count := 0
	p := NewPoller(func() bool {
		calledSig.Signal()
		count++
		return false
	}, 10*time.Millisecond)

	Do(&calledSig, func() {
		a.True(p.Stop())
	})

	// Make sure it stops polling beyond the first time.
	time.Sleep(100 * time.Millisecond)

	a.Equal(1, count)
}
