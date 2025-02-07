package phonehome

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
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
		Times(2).Do(func(any, any, ...any) { g.Stop() })

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

	lastTrack := concurrency.NewSignal()
	defer lastTrack.Wait()
	expectedTraits := matchOptions(telemeter.WithTraits(map[string]any{"key": "value"}))
	const expectedEvent = "Updated Test Identity"
	const nTimes = 4
	gomock.InOrder(
		t.EXPECT().Track(expectedEvent, nil, expectedTraits).Times(nTimes-1),
		// Stop gathering after 3rd heartbeat:
		t.EXPECT().Track(expectedEvent, nil, expectedTraits).Times(1).
			Do(func(any, any, ...any) {
				lastTrack.Signal()
			}))
	g := newGatherer("Test", t, 24*time.Hour)
	defer g.Stop()
	tickChan := make(chan time.Time)
	defer close(tickChan)
	g.tickerFactory = func(time.Duration) *time.Ticker {
		return &time.Ticker{C: tickChan}
	}
	n := make(chan int64)
	defer close(n)
	var i atomic.Int64
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		n <- i.Add(1)
		return map[string]any{"key": "value"}, nil
	})
	g.Start()
	s.Equal(int64(1), <-n, "gathering should be called once on start")
	for i := 2; i <= nTimes; i++ {
		tickChan <- time.Now()
		s.Equal(int64(i), <-n, "gathering should be called on tick")
	}
}

func (s *gathererTestSuite) TestGathererWithNoDuplicates() {
	t := mocks.NewMockTelemeter(gomock.NewController(s.T()))

	lastTrack := concurrency.NewSignal()
	defer lastTrack.Wait()
	expectedTraits := matchOptions(
		telemeter.WithNoDuplicates("abc"),
		telemeter.WithTraits(map[string]any{"key": "value"}),
	)
	const expectedEvent = "Updated Test Identity"
	t.EXPECT().Track(expectedEvent, nil, expectedTraits).Times(1).
		Do(func(any, any, ...any) {
			lastTrack.Signal()
		})
	g := newGatherer("Test", t, 24*time.Hour)
	defer g.Stop()
	n := make(chan int64)
	defer close(n)
	var i atomic.Int64
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		n <- i.Add(1)
		return map[string]any{"key": "value"}, nil
	})
	g.Start(func(co *telemeter.CallOptions) {
		telemeter.WithNoDuplicates("abc")(co)
	})
	s.Equal(int64(1), <-n, "gathering should be called once on start")
	// The cache is implemented by the telemeter, and is out of scope here, as
	// the mock is used. So we don't test the actual deduplication here.
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
