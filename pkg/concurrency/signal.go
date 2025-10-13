package concurrency

import (
	"sync/atomic"
	"unsafe"
)

// ReadOnlySignal provides an interface to inspect a Signal without modifying it.
type ReadOnlySignal interface {
	WaitC() WaitableChan
	Done() <-chan struct{}
	IsDone() bool
	Wait()
	Snapshot() WaitableChan
}

// Signal implements a signalling facility. Unlike sync.Cond, it is based on channels and can hence be used
// in `select` statements.
// There are two ways to instantiate a Signal. The preferred way is by calling `NewSignal()`, which will return a signal
// that is not triggered. Alternatively, the zero-value can be used to instantiate a signal in triggered condition,
// which is not what you usually want. To reset it to the non-triggered state, call `Reset()`.
// Similarly to `sync.(RW)Mutex` and `sync.Cond`, a signal should not be copied once used.
type Signal struct {
	ch unsafe.Pointer // ch is a pointer to the signal channel, or `nil` if the signal is in the triggered state.
}

// NewSignal creates a new signal that is in the reset state.
func NewSignal() Signal {
	var s Signal
	s.Reset()
	return s
}

// WaitC returns a WaitableChan for this signal.
func (s *Signal) WaitC() WaitableChan {
	chPtr := atomic.LoadPointer(&s.ch)
	if chPtr == nil {
		return closedCh
	}

	ch := (*chan struct{})(chPtr)
	return *ch
}

// Done returns a channel that is closed when this signal was triggered.
func (s *Signal) Done() <-chan struct{} {
	return s.WaitC()
}

// IsDone checks if the signal was triggered. It is a slightly more efficient alternative to calling `IsDone(s)`.
func (s *Signal) IsDone() bool {
	chPtr := atomic.LoadPointer(&s.ch)
	if chPtr == nil {
		return true
	}
	ch := (*chan struct{})(chPtr)
	return IsDone(WaitableChan(*ch))
}

// Wait waits for the signal to be triggered. It is a slightly more efficient and convenient alternative to calling
// `Wait(s)`.
func (s *Signal) Wait() {
	chPtr := atomic.LoadPointer(&s.ch)
	if chPtr == nil {
		return
	}
	ch := (*chan struct{})(chPtr)
	Wait(WaitableChan(*ch))
}

// Reset resets the signal to the non-triggered state, if necessary. The return value indicates whether a reset was
// actually performed (i.e., the signal was triggered). It returns false if the signal was not in the triggered state.
func (s *Signal) Reset() bool {
	ch := make(chan struct{})
	//#nosec G103
	return atomic.CompareAndSwapPointer(&s.ch, nil, unsafe.Pointer(&ch))
}

// Signal triggers the signal. The return value indicates whether the signal was actually triggered. It returns false
// if the signal was already in the triggered state.
func (s *Signal) Signal() bool {
	chPtr := atomic.SwapPointer(&s.ch, nil)
	if chPtr == nil {
		return false
	}

	ch := (*chan struct{})(chPtr)
	close(*ch)
	return true
}

// Snapshot returns a WaitableChan that observers will only see triggering once, i.e., if this signal is triggered (or
// has been triggered) and then `Reset()` is called, subsequent calls to `Done()` on the returned object will still see
// a triggered channel.
func (s *Signal) Snapshot() WaitableChan {
	return s.Done()
}

// SignalWhen triggers this signal when the given trigger condition is satisfied. It returns as soon as either this
// signal is triggered (either by this function or another goroutine), or cancelCond is triggered (in which case the
// signal will not be triggered).
// CAREFUL: This function blocks; if you do not want this, invoke it in a goroutine.
func (s *Signal) SignalWhen(triggerCond Waitable, cancelCond Waitable) bool {
	select {
	case <-triggerCond.Done():
		return s.Signal()
	case <-cancelCond.Done():
		return false
	case <-s.Done():
		return false
	}
}
