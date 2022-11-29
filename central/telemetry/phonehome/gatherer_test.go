package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

type gathererTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
}

var _ suite.SetupTestSuite = (*gathererTestSuite)(nil)

func (s *gathererTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(&testing.T{})
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

func (s *gathererTestSuite) TestNilGatherer() {
	s.T().Setenv(env.TelemetryStorageKey.EnvVar(), "")
	nilgatherer := GathererSingleton()
	s.Nil(nilgatherer)
	nilgatherer.Start() // noop
	nilgatherer.Stop()  // noop
}

func (s *gathererTestSuite) TestGatherer() {
	s.T().Setenv(env.TelemetryStorageKey.EnvVar(), "testkey")

	var i int64
	stop := concurrency.NewSignal()
	gptr := newGatherer(nil, 1*time.Second, func(context.Context) (map[string]any, error) {
		if atomic.AddInt64(&i, 1) > 1 {
			stop.Signal()
		}
		return nil, nil
	})
	go func() {
		stop.Wait()
		gptr.Stop()
	}()
	s.NotNil(gptr)
	gptr.Start()

	<-gptr.ctx.Done()
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)

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
	s.Equal(int64(3), i)
}

func (s *gathererTestSuite) TestAddTotal() {
	m := make(map[string]any)
	addTotal(context.Background(), m, "key", func(ctx context.Context) ([]*string, error) {
		return []*string{}, nil
	})
	s.Equal(0, m["Total key"])

	addTotal(context.Background(), m, "key1", func(ctx context.Context) ([]*string, error) {
		one := ""
		return []*string{&one}, nil
	})
	s.Equal(1, m["Total key1"])
}
