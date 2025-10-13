package concurrency

import (
	"context"
	"time"
)

// DependentContext creates a cancellable context that is a child of parent context, and will be cancelled if the given
// signal is triggered.
func DependentContext(parentCtx context.Context, signal Waitable) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parentCtx)
	CancelContextOnSignal(ctx, cancel, signal)
	return ctx, cancel
}

// CancelContextOnSignal cancels the given context (by invoking it's cancellation function) if the given signal is
// triggered. The context itself needs to be specified to ensure that the goroutine spawned by this functon does not
// outlive the context.
func CancelContextOnSignal(ctx context.Context, cancel context.CancelFunc, signal Waitable) {
	go func() {
		select {
		case <-ctx.Done():
		case <-signal.Done():
			cancel()
		}
	}()
}

type contextWrapper struct {
	ErrorWaitable
	isDone func() bool
}

func (w *contextWrapper) IsDone() bool {
	if w.isDone != nil {
		return w.isDone()
	}
	select {
	case <-w.Done():
		return true
	default:
		return false
	}
}

func (w *contextWrapper) Err() error {
	if !w.IsDone() {
		return nil
	}
	err := w.ErrorWaitable.Err()
	if err != context.DeadlineExceeded {
		return context.Canceled
	}
	return err
}

func (w *contextWrapper) Value(_ interface{}) interface{} {
	return nil
}

func (w *contextWrapper) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

// AsContext returns a wrapper object that makes the given waitable appear like a context without values.
func AsContext(w Waitable) context.Context {
	if ctx, _ := w.(context.Context); ctx != nil {
		return ctx
	}
	var isDone func() bool
	if supportsIsDone, _ := w.(interface{ IsDone() bool }); supportsIsDone != nil {
		isDone = supportsIsDone.IsDone
	}

	return &contextWrapper{
		ErrorWaitable: AsErrorWaitable(w),
		isDone:        isDone,
	}
}
