package concurrency

import (
	"github.com/stackrox/rox/pkg/sync"
)

// WaitGroup is an improved implementation of sync.WaitGroup, with similar semantics, but allowing to wait for the
// event that the counter drops to zero (or below) by means of a channel, implementing the `Waitable` interface.
// The default-constructed instance of this is a wait group with a count set to zero, meaning that `Done()` will return
// a closed channel.
type WaitGroup struct {
	cond    chan struct{}
	counter int
	mutex   sync.Mutex
}

// NewWaitGroup returns a new waitgroup instance with the given initial count.
func NewWaitGroup(initialCounter int) WaitGroup {
	var cond chan struct{}
	if initialCounter > 0 {
		cond = make(chan struct{})
	}
	return WaitGroup{
		cond:    cond,
		counter: initialCounter,
	}
}

// Reset sets the counter of the waitgroup to the given value. Any listeners will be notified if this causes the
// counter to cross the zero threshold.
func (w *WaitGroup) Reset(newCount int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.cond != nil && newCount <= 0 {
		close(w.cond)
		w.cond = nil
	} else if w.cond == nil && newCount > 0 {
		w.cond = make(chan struct{})
	}
	w.counter = newCount
}

// Add adds the given delta to the counter value. Note that this also doubles as the `Done()` method of
// `sync.WaitGroup`, by supplying a value of -1.
func (w *WaitGroup) Add(delta int) {
	if delta == 0 {
		return
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.counter += delta
	if w.cond == nil && w.counter > 0 {
		w.cond = make(chan struct{})
	} else if w.cond != nil && w.counter <= 0 {
		close(w.cond)
		w.cond = nil
	}
}

// Done returns the channel indicating the event of the counter dropping to (or below) zero.
func (w *WaitGroup) Done() <-chan struct{} {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.cond == nil {
		return closedCh
	}
	return w.cond
}
