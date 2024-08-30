package concurrency

var (
	closedCh = func() chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	}()
)
