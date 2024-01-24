package concurrency

import (
	"context"
	"time"
)

//lint:file-ignore ST1008 We do want to return errors as the first value here.

// CheckError returns true as the second return value if the given ErrorWaitable was in the triggered state, along with
// the respective error (which might be nil). If the ErrorWaitable was not triggered, `nil, false` is returned.
// CAVEAT: This function is not safe to be used if concurrent goroutines might reset the underlying waitable. When using
// this function with an `ErrorSignal` and this might happen, obtain a snapshot beforehand.
func CheckError(ew ErrorWaitable) (Error, bool) {
	select {
	case <-ew.Done():
		return ew.Err(), true
	default:
		return nil, false
	}
}

// ErrorWithDefault returns the error of the given, triggered ErrorWaitable, or a `defaultErr` if the error is non-nil.
// If the ErrorWaitable is not triggered, nil is always returned.
func ErrorWithDefault(ew ErrorWaitable, defaultErr error) error {
	err, ok := CheckError(ew)
	if ok {
		if err == nil {
			err = defaultErr
		}
	}
	return err
}

// WaitForError unconditionally waits until the given error waitable is triggered, and returns its error, if any.
func WaitForError(ew ErrorWaitable) Error {
	<-ew.Done()
	return ew.Err()
}

// WaitForErrorUntil waits until the given error waitable is triggered, in which case the return value is the same as
// for `CheckError(ew)`. If cancelCond is triggered before ew is triggered, `nil, false` is returned.
func WaitForErrorUntil(ew ErrorWaitable, cancelCond Waitable) (Error, bool) {
	select {
	case <-ew.Done():
		return ew.Err(), true
	case <-cancelCond.Done():
		return nil, false
	}
}

// WaitForErrorInContext waits until the given ErrorWaitable is triggered, in which case the return value is the same
// as for CheckError. If the parent Waitable is triggered before the ErrorWaitable is triggered, nil is returned.
func WaitForErrorInContext(ew ErrorWaitable, parentContext Waitable) Error {
	select {
	case <-ew.Done():
		return ew.Err()
	case <-parentContext.Done():
		return nil
	}
}

// WaitForErrorWithTimeout waits for the given ErrorWaitable and returns the error (equivalent to `CheckError(ew)`) once
// this happens. If the given timeout expires beforehand, `nil, false` is returnd.
func WaitForErrorWithTimeout(ew ErrorWaitable, timeout time.Duration) (Error, bool) {
	if timeout <= 0 {
		return CheckError(ew)
	}

	return WaitForErrorUntil(ew, TimeoutOr(timeout, ew))
}

// WaitForErrorWithDeadline is equivalent to `WaitForErrorWithTimeout(ew, time.Until(deadline))`.
func WaitForErrorWithDeadline(ew ErrorWaitable, deadline time.Time) (Error, bool) {
	return WaitForErrorWithTimeout(ew, time.Until(deadline))
}

// ErrorC returns a channel that can be used to receive the error from the given error waitable. If ew is triggered with
// a nil error, this nil error is sent to the channel nonetheless. However, is cancelCond is triggered before ew is,
// the channel will be closed without ever sending a value.
func ErrorC(ew ErrorWaitable, cancelCond Waitable) <-chan error {
	errC := make(chan error, 1)

	go func() {
		select {
		case <-ew.Done():
			errC <- ew.Err()
		case <-cancelCond.Done():
		}
		close(errC)
	}()
	return errC
}

type errorNow struct {
	err error
}

func (e errorNow) Err() error {
	return e.err
}

func (e errorNow) Done() <-chan struct{} {
	return closedCh
}

// ErrorNow returns an `ErrorWaitable` that is always triggered and returns the given error.
func ErrorNow(err error) ErrorWaitable {
	return errorNow{
		err: err,
	}
}

type errorWaitableWrapper struct {
	w       Waitable
	isDone  func() bool
	doneErr error
}

func (w *errorWaitableWrapper) Err() error {
	if w.isDone() {
		return w.doneErr
	}
	return nil
}

func (w *errorWaitableWrapper) IsDone() bool {
	return w.isDone()
}

func (w *errorWaitableWrapper) Done() <-chan struct{} {
	return w.w.Done()
}

// AsErrorWaitable wraps a given waitable into an error waitable.
func AsErrorWaitable(w Waitable) ErrorWaitable {
	if ew, _ := w.(ErrorWaitable); ew != nil {
		return ew
	}
	var isDone func() bool
	if doneChecker, _ := w.(interface{ IsDone() bool }); doneChecker != nil {
		isDone = doneChecker.IsDone
	} else {
		isDone = func() bool { return IsDone(w) }
	}
	return &errorWaitableWrapper{
		w:       w,
		isDone:  isDone,
		doneErr: context.Canceled,
	}
}
