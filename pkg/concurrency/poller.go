package concurrency

import (
	"time"
)

// A Poller keeps polling until a condition is met,
// and offers functionality similar to that provided in the Signal struct
// to wait for the condition to be met.
type Poller struct {
	internal Signal
	stopped  Flag
}

func (p *Poller) run(condition func() bool, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		if p.stopped.Get() {
			return
		}
		if condition() {
			p.internal.Signal()
			p.stopped.Set(true)
			return
		}
	}
}

// NewPoller returns a poller that keeps checking the given condition at the specified interval.
// It is the caller's responsibility to stop the poller. ALWAYS "defer p.Stop()" right after creating the new poller.
func NewPoller(condition func() bool, interval time.Duration) *Poller {
	p := &Poller{
		internal: NewSignal(),
	}
	go p.run(condition, interval)
	return p
}

// Done returns a channel that is closed when the poller is done.
func (p *Poller) Done() <-chan struct{} {
	return p.internal.Done()
}

// IsDone checks if the poller is done.
func (p *Poller) IsDone() bool {
	return p.internal.IsDone()
}

// Wait waits until the condition is satisfied.
func (p *Poller) Wait() {
	p.internal.Wait()
}

// Stop stops the poller from repeatedly calling the function and releases associated resources.
// Do NOT use the Poller in any way after calling Stop; the behavior is undefined.
func (p *Poller) Stop() bool {
	return !p.stopped.TestAndSet(true)
}

// PollWithTimeout is a utility wrapper around Poller and WaitWithTimeout.
// It polls, at the duration specified by interval, until the passed condition func returns true.
// It gives up after timeout.
// It returns true if the condition was met, and false if there was a timeout.
func PollWithTimeout(condition func() bool, interval, timeout time.Duration) bool {
	p := NewPoller(condition, interval)
	defer p.Stop()
	return WaitWithTimeout(p, timeout)
}
