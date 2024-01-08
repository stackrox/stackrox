package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

	t.EXPECT().Track("Updated Test Identity", nil,
		matchOptions(telemeter.WithTraits(map[string]any{"key": "value"}))).Times(1)

	props := make(map[string]any)
	var i int64

	g := newGatherer("Test", t, 24*time.Hour)
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
