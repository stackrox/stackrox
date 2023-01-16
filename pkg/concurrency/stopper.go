package concurrency

// NewStopper creates a new Stopper for arranging a graceful shutdown.
func NewStopper() Stopper {
	return &stopperImpl{
		stop:    NewErrorSignal(),
		stopped: NewErrorSignal(),
	}
}

// Stopper encapsulates stop and stopped signals for arranging a graceful shutdown sequence for goroutines (and async
// processing in general). These are referred as "stoppable" goroutines.
type Stopper interface {
	// Client returns an interface that should be used outside the goroutine to request a graceful shutdown and check
	// the shutdown status.
	Client() StopperClient

	// Flow returns an interface that should be used in stoppable goroutine control flow to check and modify the state
	// of Stopper. Flow should not be used outside the stoppable goroutine.
	Flow() StopperFlow

	// LowLevel allows to interact with the stopper at lower level. Callers should avoid using it unless there's no
	// better alternative.
	LowLevel() StopperLowLevel
}

// StopperClient represents an interface for an external client (external to a stoppable goroutine) to request the
// stoppable goroutine to gracefully shutdown and to check the shutdown status.
type StopperClient interface {
	// Stop requests stoppable goroutine to shut down (or stop).
	Stop()

	// Stopped returns ReadOnlyErrorSignal that can be used for waiting until the stoppable goroutine reports that it
	// finally stopped.
	Stopped() ReadOnlyErrorSignal
}

// StopperFlow represents an interface that should be used by a stoppable goroutine to check and report its status of a
// graceful shutdown.
type StopperFlow interface {
	// StopWithError initiates a shutdown due to an error that happened during async processing in goroutine. Should be
	// called from the stoppable goroutine.
	StopWithError(error)

	// StopRequested provides a channel that should be checked in goroutine in select statements for goroutine to
	// recognize that the shutdown was requested, and it needs to wrap up its work.
	StopRequested() <-chan struct{}

	// ReportStopped must be used by goroutine to report that the shutdown has been completed.
	// It is best to call this as the last thing in goroutine. Consider using defer statement for this call.
	ReportStopped()
}

// StopperLowLevel is a lower level API for interacting with Stopper. Callers should avoid using it unless there's no
// better alternative.
type StopperLowLevel interface {
	// GetStopRequestSignal returns a ReadOnlyErrorSignal for the event when a stop was requested.
	GetStopRequestSignal() ReadOnlyErrorSignal

	// ResetStopRequest resets the stop request to the untriggered state. Returns true if the stop request was in the
	// triggered state at the time of the call.
	ResetStopRequest() bool
}

type stopperImpl struct {
	// stop signal is for requesting shutdown.
	stop ErrorSignal
	// stopped is for signalling that the shutdown is complete.
	stopped ErrorSignal
}

// Client and its functions.

func (s *stopperImpl) Client() StopperClient {
	return s
}

func (s *stopperImpl) Stop() {
	s.stop.Signal()
}

func (s *stopperImpl) Stopped() ReadOnlyErrorSignal {
	return &s.stopped
}

// Flow and its functions.

func (s *stopperImpl) Flow() StopperFlow {
	return s
}

func (s *stopperImpl) StopWithError(err error) {
	s.stop.SignalWithError(err)
}

func (s *stopperImpl) StopRequested() <-chan struct{} {
	return s.stop.Done()
}

func (s *stopperImpl) ReportStopped() {
	s.stopped.SignalWithError(s.stop.Err())
}

// Low level API.

func (s *stopperImpl) LowLevel() StopperLowLevel {
	return s
}

func (s *stopperImpl) GetStopRequestSignal() ReadOnlyErrorSignal {
	return &s.stop
}

func (s *stopperImpl) ResetStopRequest() bool {
	return s.stop.Reset()
}
