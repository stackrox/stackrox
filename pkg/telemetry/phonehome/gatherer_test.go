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

func TestGatherer(t *testing.T) {
	suite.Run(t, new(gathererTestSuite))
}

func (s *gathererTestSuite) TestNilGatherer() {
	nilgatherer := NewClient("", "", "").Gatherer()

	s.NotNil(nilgatherer)
	_, ok := nilgatherer.(*nilGatherer)
	s.True(ok)
	nilgatherer.AddGatherer(func(ctx context.Context) (map[string]any, error) { return nil, nil })
	nilgatherer.Start() // noop
	nilgatherer.Stop()  // noop
}

func (s *gathererTestSuite) TestGatherer() {
	t := mocks.NewMockTelemeter(gomock.NewController(s.T()))
	g := newGatherer("Test", func() telemeter.Telemeter { return t }, 24*time.Hour)

	t.EXPECT().Identify(matchOptions(telemeter.WithTraits(map[string]any{"key": "value"}))).
		Times(2)

	t.EXPECT().Track("Updated Test Identity", nil, gomock.Any()).
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

	// Track is called from a goroutine by the gatherer.
	trackSyncCh := make(chan struct{})
	trackFunc := func(string, map[string]any, ...telemeter.Option) {
		trackSyncCh <- struct{}{}
	}

	gomock.InOrder(
		// 1
		t.EXPECT().Identify(expectedTraits).Times(1),
		t.EXPECT().Track(expectedEvent, nil, gomock.Any()).Times(1).Do(trackFunc),
		// 2
		t.EXPECT().Identify(expectedTraits).Times(1),
		t.EXPECT().Track(expectedEvent, nil, gomock.Any()).Times(1).Do(trackFunc),
		// 3
		t.EXPECT().Identify(expectedTraits).Times(1),
		t.EXPECT().Track(expectedEvent, nil, gomock.Any()).Times(1).Do(trackFunc),
		// Stop gathering after 3rd heartbeat:
		t.EXPECT().Identify(expectedTraits).Times(1),
		t.EXPECT().Track(expectedEvent, nil, gomock.Any()).Times(1).
			Do(func(any, any, ...any) {
				trackSyncCh <- struct{}{}
				lastTrack.Signal()
			}))
	g := newGatherer("Test", func() telemeter.Telemeter { return t }, 24*time.Hour)
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
	// Start will send the initial identity synchronously.
	go g.Start()
	s.Equal(int64(1), <-n, "gathering should be called once on start")
	<-trackSyncCh
	for i := 2; i <= nTimes; i++ {
		tickChan <- time.Now()
		s.Equal(int64(i), <-n, "gathering should be called on tick")
		<-trackSyncCh
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
	t.EXPECT().Identify(expectedTraits).Times(1)
	t.EXPECT().Track(expectedEvent, nil, gomock.Any()).Times(1).
		Do(func(any, any, ...any) {
			lastTrack.Signal()
		})
	g := newGatherer("Test", func() telemeter.Telemeter { return t }, 24*time.Hour)
	defer g.Stop()
	n := make(chan int64)
	defer close(n)
	var i atomic.Int64
	g.AddGatherer(func(context.Context) (map[string]any, error) {
		n <- i.Add(1)
		return map[string]any{"key": "value"}, nil
	})
	go g.Start(func(co *telemeter.CallOptions) {
		telemeter.WithNoDuplicates("abc")(co)
	})
	s.Equal(int64(1), <-n, "gathering should be called once on start")
	// The cache is implemented by the telemeter, and is out of scope here, as
	// the mock is used. So we don't test the actual deduplication here.
}

func (s *gathererTestSuite) TestAddTotal() {
	props := make(map[string]any)
	failure := errors.New("test error")

	customFunc := func(ctx context.Context, c int) (int, error) {
		return c, nil
	}

	funcs := map[string]struct {
		f        TotalFunc
		expected any
		err      error
	}{
		"Constant": {
			f:        Constant(42),
			expected: 42,
		},
		"Another constant": {f: Constant(43),
			expected: 43,
		},
		"Failure": {
			f: func(context.Context) (int, error) {
				return 42, failure
			},
			expected: nil,
			err:      failure,
		},
		"Bind2nd": {
			f:        Bind2nd(customFunc)(44),
			expected: 44,
		},
		"Len": {
			f:        Len([]int{1, 2, 3, 4, 5}),
			expected: 5,
		},
	}
	for key, f := range funcs {
		err := AddTotal(context.Background(), props, key, f.f)
		s.ErrorIs(err, f.err)
		s.Equal(f.expected, props["Total "+key])
	}
}
