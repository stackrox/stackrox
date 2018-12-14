package concurrency

import (
	"sync"
	"time"
)

// Error is an alias for the builtin `error` interface. It is needed to trick the linter into
// not complaining about error not being the last return value.
type Error = error

// ErrorSignal is a signal that supports atomically storing an error whenever it is triggered. It is safe
// to trigger the signal concurrently from different goroutines, but only the first successful call to
// `SignalWithError` will result in the error being stored.
type ErrorSignal struct {
	mutex sync.RWMutex

	err     error
	signalC chan struct{}
}

// NewErrorSignal creates and returns a new error signal.
func NewErrorSignal() ErrorSignal {
	return ErrorSignal{
		signalC: make(chan struct{}),
	}
}

// WaitC returns a WaitableChan for this error signal.
func (s *ErrorSignal) WaitC() WaitableChan {
	ch := s.getC()
	if ch == nil {
		return closedCh
	}
	return ch
}

// Done returns a channel that is closed when this error signal was triggered.
func (s *ErrorSignal) Done() <-chan struct{} {
	return s.WaitC()
}

// Reset resets the error signal to the untriggered state. It returns true if the signal was in the triggered
// state and was reset, false otherwise.
func (s *ErrorSignal) Reset() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.signalC != nil {
		return false
	}
	s.err = nil
	s.signalC = make(chan struct{})
	return true
}

// Signal triggers the signal, storing a `nil` error.
func (s *ErrorSignal) Signal() bool {
	return s.SignalWithError(nil)
}

// SignalWithError triggers the signal and stores the given error. It returns true if the signal was in
// the untriggered state and the error was stored, false otherwise.
func (s *ErrorSignal) SignalWithError(err error) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.signalC == nil {
		return false
	}
	s.err = err
	close(s.signalC)
	s.signalC = nil
	return true
}

func (s *ErrorSignal) getC() <-chan struct{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.signalC
}

// IsDone checks if the error signal was triggered. It is a slightly more efficient alternative to calling
// `IsDone(s)`.
func (s *ErrorSignal) IsDone() bool {
	return s.getC() == nil
}

// Error returns the error that was stored when triggering the signal. If the signal has not been triggered,
// the second return value returns false.
func (s *ErrorSignal) Error() (Error, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.signalC == nil {
		return s.err, true
	}
	return nil, false
}

// WaitUntil waits until the signal is triggered or another condition is met. If the signal is triggered
// before the condition is met, the stored error is returned along with a `true` boolean value. Otherwise,
// a `nil` error and `false` are returned.
func (s *ErrorSignal) WaitUntil(cancelCond Waitable) (Error, bool) {
	s.mutex.RLock()
	if s.signalC == nil {
		err := s.err
		s.mutex.RUnlock()
		return err, true
	}
	ch := s.signalC
	s.mutex.RUnlock()

	select {
	case <-ch:
		return s.Error()
	case <-cancelCond.Done():
		return nil, false
	}
}

// WaitWithTimeout waits for this signal to be triggered or the timeout to expire.
func (s *ErrorSignal) WaitWithTimeout(timeout time.Duration) (Error, bool) {
	if timeout <= 0 {
		return s.Error()
	}

	s.mutex.RLock()
	if s.signalC == nil {
		err := s.err
		s.mutex.RUnlock()
		return err, true
	}
	ch := s.signalC
	s.mutex.RUnlock()

	timer := time.NewTimer(timeout)
	select {
	case <-ch:
		if !timer.Stop() {
			<-timer.C
		}
		return s.Error()
	case <-timer.C:
		return nil, false
	}
}

// WaitWithDeadline waits for this signal to be triggered or the deadline to exceed.
func (s *ErrorSignal) WaitWithDeadline(deadline time.Time) (Error, bool) {
	return s.WaitWithTimeout(deadline.Sub(time.Now()))
}

// Wait waits indefinitely for this signal to be triggered, and returns the stored error.
func (s *ErrorSignal) Wait() error {
	err, _ := s.WaitUntil(WaitableChan(nil))
	return err
}

// Err returns the error stored for this error signal. If the signal has not been triggered, nil is returned.
// Note that this does not allow you to distinguish between a triggered signal with a `nil` error and an
// untriggered error signal. Use `Error()` if this is necessary.
func (s *ErrorSignal) Err() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.err
}
