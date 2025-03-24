package phonehome

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// GatherFunc returns properties gathered by a data source.
type GatherFunc func(context.Context) (map[string]any, error)

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start(...telemeter.Option)
	Stop()
	AddGatherer(GatherFunc)
}

type nilGatherer struct{}

func (*nilGatherer) Start(...telemeter.Option) {}
func (*nilGatherer) Stop()                     {}
func (*nilGatherer) AddGatherer(GatherFunc)    {}

type gatherer struct {
	clientType  string
	telemeter   telemeter.Telemeter
	period      time.Duration
	stopSig     concurrency.Signal
	ctx         context.Context
	gathering   sync.Mutex
	gatherFuncs []GatherFunc
	opts        []telemeter.Option

	// tickerFactory allows for setting a custom ticker for ad-hoc gathering.
	tickerFactory func(time.Duration) *time.Ticker
}

func newGatherer(clientType string, t telemeter.Telemeter, p time.Duration) *gatherer {
	return &gatherer{
		clientType: clientType,
		telemeter:  t,
		period:     p,

		tickerFactory: time.NewTicker,
	}
}

func (g *gatherer) gather() map[string]any {
	var result map[string]any
	for i, f := range g.gatherFuncs {
		props, err := f(g.ctx)
		if err != nil {
			log.Errorf("gatherer %d failure: %v", i, err)
		}
		if props != nil && result == nil {
			result = make(map[string]any, len(props))
		}
		for k, v := range props {
			result[k] = v
		}
	}
	return result
}

func (g *gatherer) identify() {
	// TODO: might make sense to abort if !TryLock(), but that's harder to test.
	g.gathering.Lock()
	defer g.gathering.Unlock()
	data := g.gather()
	// Track event makes the properties effective for the user on analytics.
	// Duplicates are dropped during a day. The daily potential duplicate event
	// serves as a heartbeat.
	g.telemeter.Track("Updated "+g.clientType+" Identity", nil, append(g.opts,
		telemeter.WithTraits(data))...)
}

func (g *gatherer) loop() {
	// Send initial data on start:
	g.identify()
	ticker := g.tickerFactory(g.period)
	defer ticker.Stop()
	for !g.stopSig.IsDone() {
		select {
		case _, ok := <-ticker.C:
			if ok {
				go g.identify()
			}
		case <-g.stopSig.Done():
			return
		}
	}
}

func (g *gatherer) Start(opts ...telemeter.Option) {
	if g == nil || !g.stopSig.IsDone() {
		return
	}
	concurrency.WithLock(&g.gathering, func() {
		g.stopSig.Reset()
		g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
		g.opts = opts
	})
	go g.loop()
}

func (g *gatherer) Stop() {
	if g != nil {
		g.stopSig.Signal()
	}
}

func (g *gatherer) AddGatherer(f GatherFunc) {
	if g == nil {
		return
	}
	g.gathering.Lock()
	defer g.gathering.Unlock()
	g.gatherFuncs = append(g.gatherFuncs, f)
}

type TotalFunc func(context.Context) (int, error)

// AddTotal sets an entry in the props map with key and number returned by f as
// the value.
func AddTotal(ctx context.Context, props map[string]any, key string, f TotalFunc) error {
	ps, err := f(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s", key)
	}
	props["Total "+key] = ps
	return nil
}

// Bind2nd returns a function that allows to bind the second parameter for the
// given function f.
//
// Example:
//
//	func myfunc(_ context.Context, v int) (int, error) {
//		return v, nil
//	}
//	...
//	f := Bind2nd(myfunc)
//	bound2nd := f(42)
//	bound2nd(context.Background) === myfunc(context.Background, 42)
func Bind2nd[A any](f func(context.Context, A) (int, error)) func(A) TotalFunc {
	return func(arg A) TotalFunc {
		return func(ctx context.Context) (int, error) {
			return f(ctx, arg)
		}
	}
}

// Constant makes a TotalFunc that returns the provided constant value.
func Constant(a int) TotalFunc {
	return func(_ context.Context) (int, error) {
		return a, nil
	}
}

// Len makes a TotalFunc that computes the length of the provided slice.
func Len[T any](arr []T) TotalFunc {
	return func(_ context.Context) (int, error) {
		return len(arr), nil
	}
}
