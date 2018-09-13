package concurrency

// Turnstile is a concurrency primitive that is akin to a turnstile. When it is unlocked (AllowOne) a single waiting
// client is able to get through (Wait returns true). When disabled (Close), all clients are allowed through, but
// with the knowledge that it has been disabled (Wait returns false).
type Turnstile interface {
	AllowOne()
	Wait() bool
	WaitWithCancel(cancelWhen Waitable) bool
	Close() bool
}

// NewTurnstile returns a new instance of a Turnstile.
func NewTurnstile() Turnstile {
	return &turnstileImpl{
		flag: Flag{
			val: 0,
		},
		pendingChannel: make(chan struct{}, 1),
	}
}

type turnstileImpl struct {
	flag           Flag
	pendingChannel chan struct{}
}

// AllowOne lets one waiting go routine through.
func (s *turnstileImpl) AllowOne() {
	select {
	case s.pendingChannel <- struct{}{}:
		return
	default:
		return
	}
}

// Wait waits until a separate go routine calls AllowOne (returns true) or Close (returns false).
func (s *turnstileImpl) Wait() bool {
	_, ok := <-s.pendingChannel
	return ok
}

// WaitWithCancel waits until a separate go routine calls AllowOne (returns true), Close (returns false), or the
// cancelWhen Waitable object returns (returns false).
func (s *turnstileImpl) WaitWithCancel(cancelWhen Waitable) bool {
	select {
	case _, ok := <-s.pendingChannel:
		return ok
	case <-cancelWhen.Done():
		return false
	}
}

// Close closes the turnstile, allowing all waiting go routines through, but returning false from all 'Wait' calls.
// Returns true if the turnstile closed, and false if a separate go routine already closed it.
func (s *turnstileImpl) Close() bool {
	if s.flag.TestAndSet(true) {
		return false
	}
	close(s.pendingChannel)
	return true
}
