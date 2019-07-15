package concurrency

// Waitable is a generic interface for things that can be waited upon. The method `Done` returns a channel that, when
// closed, signals that whatever condition is represented by this waitable is satisfied.
// Note: The name `Done` was chosen such that `context.Context` conforms to this interface.
type Waitable interface {
	Done() <-chan struct{}
}

// WaitableChan is an alias around a `<-chan struct{}` that returns itself in its `Done` method.
type WaitableChan <-chan struct{}

// Done returns the channel itself.
func (c WaitableChan) Done() <-chan struct{} {
	return c
}

// ErrorWaitable is a generic interface for things that can be waited upon, and might return an error upon completion.
// This interface is designed to be compatible with `context.Context`, but is strictly more general in the sense that:
// - Err() might return nil even if the channel returned by `Done()` is closed.
// - Err() might return any error, not just `context.Canceled` and `context.DeadlineExceeded`.
type ErrorWaitable interface {
	Waitable

	// Err returns the error associated with this object. Before the channel returned by `Done()`, this should return
	// `nil`. Afterwards, it may return `nil` if the underlying signal was triggered without an error, or the respective
	// error that was encountered.
	// PLEASE NOTE: For triggers that can be reset after an error occurred, the combination of waiting on `Done()` and
	// calling `Err()` is prone to race conditions. For these objects such as `ErrorSignal` defined in this package,
	// use specific methods like `WaitUntil` that allow atomically waiting and retrieving the error state.
	Err() error
}
