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
	t := mocks.NewMockTelemeter(gomock.NewController(s.T()))
	t.EXPECT().Identify("test", nil).Times(3)

	var i int64
	stop := concurrency.NewSignal()
	gptr := newGatherer("test", t, 10*time.Millisecond)
	s.Require().NotNil(gptr)
	gptr.AddGatherer(func(context.Context) (map[string]any, error) {
		if atomic.AddInt64(&i, 1) > 1 {
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
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)
	s.Equal(int64(2), i)

	stop.Reset()
	go func() {
		stop.Wait()
		gptr.Stop()
	}()

	// Should start again.
	gptr.Start()
	<-gptr.ctx.Done()
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)
	s.Equal(int64(3), i)
}
