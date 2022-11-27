package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/mocks"
	"github.com/stretchr/testify/suite"
)

type gathererTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
}

var _ interface {
	suite.SetupTestSuite
	suite.TearDownTestSuite
} = (*gathererTestSuite)(nil)

func (s *gathererTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *gathererTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
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
	t := mocks.NewMockTelemeter(s.mockCtrl)

	var i int64
	gptr := newGatherer(t, 10*time.Millisecond, func(g *gatherer) {
		if i == 2 {
			g.Stop()
		} else {
			atomic.AddInt64(&i, 1)
		}
	})
	s.NotNil(gptr)
	gptr.Start()

	<-gptr.ctx.Done()
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)

	s.Nil(gptr.ticker)
	s.ErrorIs(gptr.ctx.Err(), context.Canceled)
	s.Equal(int64(2), i)

	// Should start again.
	gptr.Start()
	<-gptr.ctx.Done()
	s.Equal(int64(2), i)
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
