package concurrency

import (
	"context"
)

//lint:file-ignore ST1008 We do want to return errors as the first value here.

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
