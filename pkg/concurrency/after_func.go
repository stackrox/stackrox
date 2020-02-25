package concurrency

import (
	"time"
)

// AfterFunc is like time.AfterFunc, but also accepts a Waitable, and ensures that
// the AfterFunc is canceled if the waitable expires first.
// Like time.AfterFunc, it is non-blocking.
func AfterFunc(d time.Duration, f func(), w Waitable) {
	t := time.NewTimer(d)
	go func() {
		select {
		case <-w.Done():
			if !t.Stop() {
				<-t.C
			}
		case <-t.C:
			f()
		}
	}()
}
