package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/mocks"
	"github.com/stretchr/testify/suite"
)

type gathererTestSuite struct {
	suite.Suite
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

func (s *gathererTestSuite) TestNilGatherer() {
	cfg := &Config{}
	nilgatherer := cfg.Gatherer()
	s.NotNil(nilgatherer)
	nilgatherer.Start() // noop
	nilgatherer.Stop()  // noop
}

func (s *gathererTestSuite) TestGatherer() {
	/*
		The test starts a gatherer and stops gathering after 2 executions.
		Verifies that the gatherer function and Identify() have been called 2 times.
		Then it restarts the gathering and stops after 1 execution, and ensures that
		the gatherer function and Identify() have been called once.
	*/

	identifyStop := concurrency.NewSignal()

	// Counters of the calls to Identify() and gatherer function:
	var in, gn int64

	t := mocks.NewMockTelemeter(gomock.NewController(s.T()))
	t.EXPECT().Identify("test", nil).Times(3).Do(func(string, map[string]any) {
		if atomic.AddInt64(&in, 1) > 1 {
			identifyStop.Signal()
		}
	})

	stop := concurrency.NewSignal()
	gptr := newGatherer("test", t, 10*time.Millisecond)
	s.Require().NotNil(gptr)
	gptr.AddGatherer(func(context.Context) (map[string]any, error) {
		if atomic.AddInt64(&gn, 1) > 1 {
			stop.Signal()
		}
		return nil, nil
	})
	go func() {
		stop.Wait()
		gptr.Stop()
	}()
	gptr.Start()

	<-gptr.ctx.Done()
	// Wait until Idenfity() is called after gathering:
	identifyStop.Wait()

	s.ErrorIs(gptr.ctx.Err(), context.Canceled)
	s.Equal(int64(2), in)
	s.Equal(int64(2), gn)

	identifyStop.Reset()
	stop.Reset()
	go func() {
		stop.Wait()
		gptr.Stop()
	}()

	// Should start again.
	gptr.Start()
	<-gptr.ctx.Done()
	// Wait until Idenfity() is called after gathering:
	identifyStop.Wait()
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)
	s.Equal(int64(3), in)
	s.Equal(int64(3), gn)
}
