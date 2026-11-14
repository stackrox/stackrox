package backgroundworker

// SignalChannel is a non-blocking, resettable signal compatible with
// PeriodicWorker.ShortCircuit and BatchAccumulator's flush trigger.
//
// Unlike concurrency.Signal (whose Done() channel stays closed forever
// after Signal()), this uses a buffered(1) channel that is drained on
// each select iteration, making it safe for repeated use.
type SignalChannel struct {
	ch chan struct{}
}

// NewSignalChannel creates a ready-to-use signal channel.
func NewSignalChannel() *SignalChannel {
	return &SignalChannel{ch: make(chan struct{}, 1)}
}

// Signal sends a non-blocking signal. If a signal is already pending,
// the call is a no-op (signals coalesce).
func (s *SignalChannel) Signal() {
	select {
	case s.ch <- struct{}{}:
	default:
	}
}

// C returns the underlying channel for use in select statements or as
// PeriodicWorker.ShortCircuit.
func (s *SignalChannel) C() <-chan struct{} {
	return s.ch
}
