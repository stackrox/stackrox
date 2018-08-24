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
