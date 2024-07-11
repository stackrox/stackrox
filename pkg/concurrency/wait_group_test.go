package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWaitGroup_WithZeroIsDone(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	wg := NewWaitGroup(0)
	a.True(IsDone(&wg), "wait group should be done")
}

func TestNewWaitGroup_WithOneIsNotDone(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	wg := NewWaitGroup(1)
	a.False(IsDone(&wg), "wait group should not be done")
}

func TestWaitGroup_TriggerByAdd(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	wg := NewWaitGroup(1)
	a.False(IsDone(&wg))

	var succeeded bool
	goroutineDone := NewSignal()
	go func() {
		succeeded = WaitWithTimeout(&wg, 100*time.Millisecond)
		goroutineDone.Signal()
	}()

	wg.Add(-1)
	a.True(IsDone(&wg))
	a.True(WaitWithTimeout(&goroutineDone, 100*time.Millisecond))
	a.True(succeeded)
}

func TestWaitGroup_TriggerByReset(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	wg := NewWaitGroup(1)
	a.False(IsDone(&wg))

	var succeeded bool
	goroutineDone := NewSignal()
	go func() {
		succeeded = WaitWithTimeout(&wg, 100*time.Millisecond)
		goroutineDone.Signal()
	}()

	wg.Reset(-1)
	a.True(IsDone(&wg))
	a.True(WaitWithTimeout(&goroutineDone, 100*time.Millisecond))
	a.True(succeeded)
}

func TestWaitGroup_ThresholdCrossing(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	wg := NewWaitGroup(-2)
	a.True(IsDone(&wg))
	wg.Add(1) // now: -1
	a.True(IsDone(&wg))
	wg.Add(1) // now: 0
	a.True(IsDone(&wg))
	wg.Add(2) // now: 2
	a.False(IsDone(&wg))
	wg.Add(1) // now: 3
	a.False(IsDone(&wg))
	wg.Add(-1) // now: 2
	a.False(IsDone(&wg))
	wg.Add(-2) // now: 0
	a.True(IsDone(&wg))
}
