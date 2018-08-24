package concurrency

var (
	closedCh = func() <-chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	}()
)

// ClosedChannel returns a struct{} channel that is closed.
func ClosedChannel() <-chan struct{} {
	return closedCh
}
