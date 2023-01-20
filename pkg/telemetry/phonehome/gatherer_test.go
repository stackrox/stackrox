package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
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

	// Identify and Track should be called once as there's no change in the
	// identity:
	t.EXPECT().Identify("test", gomock.Any()).Times(1)
	t.EXPECT().Track("Updated Identity", "test", nil).Times(1)

	props := make(map[string]any)
	var i int64

	g := newGatherer("test", t, 24*time.Hour)
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		atomic.AddInt64(&i, 1)
		props["key"] = "value"
		g.Stop()
		return props, nil
	})
	g.Start()
	<-g.ctx.Done()
	s.Equal("value", props["key"], "the gathering function should have been called")
	s.Equal(int64(1), i)
	g.Start()
	<-g.ctx.Done()
	s.Equal("value", props["key"], "the gathering function should have been called")
	s.Equal(int64(2), i)
}
