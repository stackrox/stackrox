package concurrency

import "context"

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
