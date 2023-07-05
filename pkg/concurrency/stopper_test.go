package concurrency

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type StopperTestSuite struct {
	suite.Suite
}

func TestStopper(t *testing.T) {
	suite.Run(t, &StopperTestSuite{})
}

// TestCommonCase demonstrates how to use Stopper for implementing a gracefully stoppable goroutine.
func (s *StopperTestSuite) TestCommonCase() {
	stopper := NewStopper()

	s1 := NewSignal()
	s2 := NewSignal()

	// This is how to implement a gracefully stoppable goroutine.
	go func() {
		// The goroutine must report that it shut down at the end.
		defer stopper.Flow().ReportStopped()
		for {
			select {
			// This is how the goroutine finds out that it needs to stop.
			case <-stopper.Flow().StopRequested():
				return
			// Simulate some work, e.g. reading from a channel.
			case <-s1.Done():
				// Here the goroutine will do its useful work. This will just unblock the test to proceed to the shutdown.
				s2.Signal()
			}
		}
	}()

	s1.Signal()
	s2.Wait() // Wait until it is time to shut down the goroutine.

	// This is how to tell goroutine to stop.
	stopper.Client().Stop()

	// This is how to wait until the goroutine actually stops.
	s.NoError(stopper.Client().Stopped().Wait())
}

// TestMultipleGoRoutinesEndOnSameStopper demonstrates how same stopper can be used to return from go routines
func (s *StopperTestSuite) TestMultipleGoRoutinesEndOnSameStopper() {
	stopper := NewStopper()

	s1 := NewSignal()
	s2 := NewSignal()
	s3 := NewSignal()
	s4 := NewSignal()

	// This is how to implement a gracefully stoppable goroutine.
	go func() {
		// The goroutine must report that it shut down at the end.
		defer stopper.Flow().ReportStopped()
		defer fmt.Println("Routine 1 stopping")
		for {
			select {
			// This is how the goroutine finds out that it needs to stop.
			case <-stopper.Flow().StopRequested():
				return
			// Simulate some work, e.g. reading from a channel.
			case <-s1.Done():
				// Here the goroutine will do its useful work. This will just unblock the test to proceed further.
				s2.Signal()
			}
		}
	}()

	go func() {
		// The goroutine must report that it shut down at the end.
		defer stopper.Flow().ReportStopped()
		defer fmt.Println("Routine 2 stopping")
		for {
			select {
			// This is how the goroutine finds out that it needs to stop.
			case <-stopper.Flow().StopRequested():
				return
			// Simulate some work, e.g. reading from a channel.
			case <-s3.Done():
				// Here the goroutine will do its useful work. This will unblock the test to proceed to shutdown.
				s4.Signal()
			}
		}
	}()

	s1.Signal()
	s2.Wait()

	s3.Signal()
	s4.Wait() // Wait until it is time to shut down the goroutines.

	// This is how to tell goroutines to stop.
	stopper.Client().Stop()

	// This is how to wait until the goroutine actually stops.
	s.NoError(stopper.Client().Stopped().Wait())
}

// TestGoroutineError demonstrates how goroutine can stop itself on error and report this error.
func (s *StopperTestSuite) TestGoroutineError() {
	stopper := NewStopper()

	ch := make(chan struct{})
	close(ch)

	go func() {
		// stopper.Flow().ReportStopped() will make sure to propagate the error to be returned by
		// stopper.Client().Stopped().Wait().
		defer stopper.Flow().ReportStopped()
		for {
			select {
			case <-stopper.Flow().StopRequested():
				return
			case _, ok := <-ch:
				// The channel starts as closed by the way how the test is set up. This wouldn't be a case in normal
				// code.
				s.False(ok)
				if !ok {
					stopper.Flow().StopWithError(errors.New("test input channel was closed"))
					return
				}
			}
		}
	}()

	// Not calling stopper.Client().Stop() because the goroutine will stop by itself on error.

	s.ErrorContains(stopper.Client().Stopped().Wait(), "test input channel was closed")
}

// TestGoroutineErrorPreserved verifies that the error reported by goroutine is preserved even in case the external
// client requests it to stop. Note that who calls Stop* first is subject of race conditions therefore this test uses
// a couple of signals to make sure StopWithError() inside goroutine is called before Stop() outside it.
func (s *StopperTestSuite) TestGoroutineErrorPreserved() {
	stopper := NewStopper()

	s1 := NewSignal()
	s2 := NewSignal()

	go func() {
		defer stopper.Flow().ReportStopped()
		stopper.Flow().StopWithError(errors.New("test failure"))
		s1.Signal()
		s2.Wait()
	}()

	s1.Wait()
	stopper.Client().Stop()
	s2.Signal() // This makes sure that stopper.Client().Stop() is called before stopper.Flow().ReportStopped().

	s.ErrorContains(stopper.Client().Stopped().Wait(), "test failure")
}

// TestOnlyStop verifies (and demonstrates) the case found in sensor/common/config/handler.go where the code relies
// only on signalling stopperImpl.stop without touching stopperImpl.stopped.
// Although Stopper can be used this way, it isn't probably a good idea to use it. The purpose of Stopper is to help
// have standard protocol of the graceful shutdown whereas a single event is not enough for that.
func (s *StopperTestSuite) TestOnlyStop() {
	stopper := NewStopper()
	action := func() error {
		select {
		case <-stopper.Flow().StopRequested():
			return errors.New("no action - stop was requested")
		default:
			return nil
		}
	}

	s.NoError(action())
	stopper.Client().Stop()
	s.ErrorContains(action(), "no action - stop was requested")
}
