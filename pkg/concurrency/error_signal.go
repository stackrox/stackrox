package concurrency

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

//lint:file-ignore ST1008 We do want to return errors as the first value here.

// Error is an alias for the builtin `error` interface. It is needed to trick the linter into
// not complaining about error not being the last return value.
type Error = error

// ReadOnlyErrorSignal provides an interface to inspect ErrorSignals without modifying them.
type ReadOnlyErrorSignal interface {
	WaitC() WaitableChan
	Done() <-chan struct{}
	IsDone() bool
	Error() (Error, bool)
	ErrorWithDefault(defaultErr error) error
	WaitUntil(cancelCond Waitable) (Error, bool)
	WaitWithTimeout(timeout time.Duration) (Error, bool)
	WaitWithDeadline(deadline time.Time) (Error, bool)
	Wait() error
	Err() error
	Snapshot() ReadOnlyErrorSignal
}

type errorSignalState struct {
	errPtr  unsafe.Pointer
	signalC chan struct{}
}

func (s *errorSignalState) Err() error {
	errP := s.getErrPtr()
	if errP == nil {
		return nil
	}
	return *errP
}

func (s *errorSignalState) IsDone() bool {
	return s.getErrPtr() != nil
}

func (s *errorSignalState) Done() <-chan struct{} {
	return s.signalC
}

func (s *errorSignalState) WaitC() WaitableChan {
	return s.Done()
}

func (s *errorSignalState) Error() (Error, bool) {
	if errPtr := s.getErrPtr(); errPtr != nil {
		return *errPtr, true
	}
	return nil, false
}

func (s *errorSignalState) ErrorWithDefault(defaultErr error) error {
	err, ok := s.Error()
	if ok && err == nil {
		err = defaultErr
	}
	return err
}

func (s *errorSignalState) WaitUntil(cancelCond Waitable) (Error, bool) {
	select {
	case <-s.signalC:
		return s.Error()
	case <-cancelCond.Done():
		return nil, false
	}
}

func (s *errorSignalState) WaitWithTimeout(timeout time.Duration) (Error, bool) {
	if timeout <= 0 {
		return s.Error()
	}

	timer := time.NewTimer(timeout)
	select {
	case <-s.signalC:
		if !timer.Stop() {
			<-timer.C
		}
		return s.Error()
	case <-timer.C:
		return nil, false
	}
}

func (s *errorSignalState) WaitWithDeadline(deadline time.Time) (Error, bool) {
	return s.WaitWithTimeout(time.Until(deadline))
}

func (s *errorSignalState) Wait() error {
	err, _ := s.WaitUntil(Never())
	return err
}

func (s *errorSignalState) Snapshot() ReadOnlyErrorSignal {
	return s
}

func (s *errorSignalState) getErrPtr() *error {
	return (*error)(atomic.LoadPointer(&s.errPtr))
}

func (s *errorSignalState) trigger(err error) bool {
	if !atomic.CompareAndSwapPointer(&s.errPtr, nil,
		//#nosec G103
		unsafe.Pointer(&err)) {
		return false
	}
	close(s.signalC)
	return true
}

func newErrorSignalState() *errorSignalState {
	return &errorSignalState{
		signalC: make(chan struct{}),
	}
}

var (
	defaultErrorSignalState = &errorSignalState{
		//#nosec G103
		errPtr:  unsafe.Pointer(&[]error{nil}[0]),
		signalC: closedCh,
	}
)

// ErrorSignal is a signal that supports atomically storing an error whenever it is triggered. It is safe
// to trigger the signal concurrently from different goroutines, but only the first successful call to
// `SignalWithError` will result in the error being stored.
type ErrorSignal struct {
	statePtr unsafe.Pointer
}

// NewErrorSignal creates and returns a new error signal.
func NewErrorSignal() ErrorSignal {
	return ErrorSignal{
		//#nosec G103
		statePtr: unsafe.Pointer(newErrorSignalState()),
	}
}

// WaitC returns a WaitableChan for this error signal.
func (s *ErrorSignal) WaitC() WaitableChan {
	return s.getStateOrDefault().Done()
}

// Done returns a channel that is closed when this error signal was triggered.
func (s *ErrorSignal) Done() <-chan struct{} {
	return s.WaitC()
}

// Reset resets the error signal to the untriggered state. It returns true if the signal was in the triggered
// state and was reset, false otherwise.
func (s *ErrorSignal) Reset() bool {
	_, success := s.ErrorAndReset()
	return success
}

// ErrorAndReset resets the error signal. If it was in the triggered state, the last signalled error (which may be nil)
// is fetched atomically before the reset happens and returned. If the signal was not triggered, `nil, false` is
// returned.
// It is guaranteed that for any number of concurrent callers to `ErrorAndReset` (and no triggering happening at the
// same time), exactly one caller will see a `true` return value with a potentially non-nil error.
func (s *ErrorSignal) ErrorAndReset() (Error, bool) {
	rawState := s.getState()
	state := rawState
	if state == nil {
		state = defaultErrorSignalState
	}

	if !state.IsDone() {
		// If we get triggered after the above fetch, that's okay - we pretend the Reset() (which then is a no-op)
		// happened strictly before the trigger.
		return nil, false
	}

	// The reset only takes place if the above, triggered state is still the current state. This ensure that if a
	// concurrent reset happened and succeeded, this Reset invocation will not. If the signal has been reset and
	// triggered in the meantime, we fail, too, pretending this Reset invocation happened as the first action in a
	// Reset - Trigger - Reset sequence.
	if !atomic.CompareAndSwapPointer(&s.statePtr,
		//#nosec G103
		unsafe.Pointer(rawState),
		//#nosec G103
		unsafe.Pointer(newErrorSignalState())) {
		return nil, false
	}
	return state.Err(), true
}

// Signal triggers the signal, storing a `nil` error.
func (s *ErrorSignal) Signal() bool {
	return s.SignalWithError(nil)
}

// SignalWithErrorWrap is a wrapper around SignalWithError and errors.Wrap.
func (s *ErrorSignal) SignalWithErrorWrap(err error, message string) bool {
	return s.SignalWithError(errors.Wrap(err, message))
}

// SignalWithErrorWrapf is a wrapper around SignalWithError and errors.Wrapf.
func (s *ErrorSignal) SignalWithErrorWrapf(err error, format string, args ...interface{}) bool {
	return s.SignalWithError(errors.Wrapf(err, format, args...))
}

// SignalWithErrorf is a wrapper around SignalWithError and fmt.Errorf.
func (s *ErrorSignal) SignalWithErrorf(format string, args ...interface{}) bool {
	return s.SignalWithError(fmt.Errorf(format, args...))
}

// SignalWithError triggers the signal and stores the given error. It returns true if the signal was in
// the untriggered state and the error was stored, false otherwise.
func (s *ErrorSignal) SignalWithError(err error) bool {
	return s.getStateOrDefault().trigger(err)
}

func (s *ErrorSignal) getState() *errorSignalState {
	return (*errorSignalState)(atomic.LoadPointer(&s.statePtr))
}

func (s *ErrorSignal) getStateOrDefault() *errorSignalState {
	st := s.getState()
	if st != nil {
		return st
	}
	return defaultErrorSignalState
}

// IsDone checks if the error signal was triggered. It is a slightly more efficient alternative to calling
// `IsDone(s)`.
func (s *ErrorSignal) IsDone() bool {
	return s.getStateOrDefault().IsDone()
}

// Error returns the error that was stored when triggering the signal. If the signal has not been triggered,
// the second return value returns false.
func (s *ErrorSignal) Error() (Error, bool) {
	return s.getStateOrDefault().Error()
}

// ErrorWithDefault returns the error that was stored when triggering the signal or, when the signal was triggered with
// a nil error, the given `defaultErr`. If the signal has not been triggered, nil is returned.
func (s *ErrorSignal) ErrorWithDefault(defaultErr error) error {
	return s.getStateOrDefault().ErrorWithDefault(defaultErr)
}

// WaitUntil waits until the signal is triggered or another condition is met. If the signal is triggered
// before the condition is met, the stored error is returned along with a `true` boolean value. Otherwise,
// a `nil` error and `false` are returned.
func (s *ErrorSignal) WaitUntil(cancelCond Waitable) (Error, bool) {
	return s.getStateOrDefault().WaitUntil(cancelCond)
}

// WaitWithTimeout waits for this signal to be triggered or the timeout to expire.
func (s *ErrorSignal) WaitWithTimeout(timeout time.Duration) (Error, bool) {
	return s.getStateOrDefault().WaitWithTimeout(timeout)
}

// WaitWithDeadline waits for this signal to be triggered or the deadline to exceed.
func (s *ErrorSignal) WaitWithDeadline(deadline time.Time) (Error, bool) {
	return s.WaitWithTimeout(time.Until(deadline))
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
	return s.getStateOrDefault().Err()
}

// Snapshot returns a read-only error signal that will not be affected by subsequent calls to `Reset`. It can be used
// to wait for the signal to be triggered and retrieve the error safely, without an intermittent call to `Reset`
// hiding the error.
func (s *ErrorSignal) Snapshot() ReadOnlyErrorSignal {
	return s.getStateOrDefault()
}

// SignalWhen triggers this signal when the given trigger condition is satisfied. It returns as soon as either this
// signal is triggered (either by this function or another goroutine), or cancelCond is triggered (in which case the
// signal will not be triggered).
// CAREFUL: This function blocks; if you do not want this, invoke it in a goroutine.
func (s *ErrorSignal) SignalWhen(triggerCond Waitable, cancelCond Waitable) bool {
	select {
	case <-triggerCond.Done():
		if triggerCondErr, ok := triggerCond.(ErrorWaitable); ok {
			return s.SignalWithError(triggerCondErr.Err())
		}
		return s.Signal()
	case <-cancelCond.Done():
		return false
	case <-s.Done():
		return false
	}
}

// SignalWithErrorWhen triggers this signal with a specified error when the given trigger condition is satisfied. It
// returns as soon as either this signal is triggered (either by this function or another goroutine), or cancelCond is
// triggered (in which case the signal will not be triggered).
// CAREFUL: This function blocks; if you do not want this, invoke it in a goroutine.
func (s *ErrorSignal) SignalWithErrorWhen(err error, triggerCond Waitable, cancelCond Waitable) bool {
	select {
	case <-triggerCond.Done():
		return s.SignalWithError(err)
	case <-cancelCond.Done():
		return false
	case <-s.Done():
		return false
	}
}
