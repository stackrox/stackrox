package concurrency

// NewStopper creates a new Stopper interface
func NewStopper() Stopper {
	return Stopper{
		stop:    NewSignal(),
		stopped: NewSignal(),
	}
}

// Stopper is an object that encapsulates both stop and stopped signals
type Stopper struct {
	stop    Signal
	stopped Signal
}

// Stop signals the internal stop signal and effectively causes a push to StopDone
func (s *Stopper) Stop() {
	s.stop.Signal()
}

// StopDone provides a channel to be used in select statements to interrupt the go routine
func (s *Stopper) StopDone() <-chan struct{} {
	return s.stop.Done()
}

// Stopped signals that the stop has been completed
func (s *Stopper) Stopped() {
	s.stopped.Signal()
}

// WaitForStopped waits for the stopped signal to be fired
func (s *Stopper) WaitForStopped() {
	s.stopped.Wait()
}
