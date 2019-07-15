package concurrency

import "time"

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
