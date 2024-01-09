package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
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
	g := newGatherer("Test", t, 24*time.Hour)

	t.EXPECT().Track("Updated Test Identity", nil,
		matchOptions(telemeter.WithTraits(map[string]any{"key": "value"}))).
		Times(1).Do(func(any, any, ...any) { g.Stop() })
	t.EXPECT().Track("Test Heartbeat", nil,
		matchOptions(telemeter.WithTraits(nil))).
		Times(1).Do(func(any, any, ...any) { g.Stop() })

	props := make(map[string]any)
	var i atomic.Int64

	g.AddGatherer(func(context.Context) (map[string]any, error) {
		i.Add(1)
		props["key"] = "value"
		return props, nil
	})
	g.Start()
	<-g.ctx.Done()
	s.Equal("value", props["key"], "the gathering function should have been called")
	s.Equal(int64(1), i.Load())
	g.Start()
	<-g.ctx.Done()
	s.Equal("value", props["key"], "the gathering function should have been called")
	s.Equal(int64(2), i.Load())
}

func (s *gathererTestSuite) TestGathererTicker() {
	t := mocks.NewMockTelemeter(gomock.NewController(s.T()))

	t.EXPECT().Track("Updated Test Identity", nil,
		matchOptions(telemeter.WithTraits(map[string]any{"key": "value"}))).Times(1)
	t.EXPECT().Track("Test Heartbeat", nil,
		matchOptions(telemeter.WithTraits(nil))).Times(3)

	g := newGatherer("Test", t, 24*time.Hour)
	tickChan := make(chan time.Time)
	g.tickerFactory = func(time.Duration) *time.Ticker {
		return &time.Ticker{C: tickChan}
	}
	n := make(chan int64)
	var i atomic.Int64
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		n <- i.Add(1)
		return map[string]any{"key": "value"}, nil
	})
	g.Start()
	s.Equal(int64(1), <-n)
	tickChan <- time.Now()
	s.Equal(int64(2), <-n)
	tickChan <- time.Now()
	tickChan <- time.Now()
	g.Stop()
	s.Equal(int64(3), <-n)
	s.Equal(int64(4), <-n)
	close(n)
}

func (s *gathererTestSuite) TestAddTotal() {
	props := make(map[string]any)
	err := AddTotal(context.Background(), props, "key 1", func(context.Context) (int, error) {
		return 42, nil
	})
	s.NoError(err)
	s.Equal(42, props["Total key 1"])

	err = AddTotal(context.Background(), props, "key 2", func(context.Context) (int, error) {
		return 43, nil
	})
	s.NoError(err)
	s.Equal(43, props["Total key 2"])

	failure := errors.New("test error")
	err = AddTotal(context.Background(), props, "key 3", func(context.Context) (int, error) {
		return 42, failure
	})
	s.ErrorIs(err, failure)
}
