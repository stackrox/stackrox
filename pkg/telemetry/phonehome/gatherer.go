package phonehome

import (
	"context"
	"reflect"
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
	mu          sync.Mutex
	gathering   sync.Mutex
	gatherFuncs []GatherFunc
	lastData    map[string]any
	opts        []telemeter.Option
}

func newGatherer(clientType string, t telemeter.Telemeter, p time.Duration) *gatherer {
	return &gatherer{
		clientType: clientType,
		telemeter:  t,
		period:     p,
	}
}

func (g *gatherer) reset() {
	g.stopSig.Reset()
	g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
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
	if !reflect.DeepEqual(g.lastData, data) {
		// Issue an event so that the new data become visible on analytics:
		g.telemeter.Track("Updated "+g.clientType+" Identity", nil, append(g.opts, telemeter.WithTraits(data))...)
	}
	g.lastData = data
}

func (g *gatherer) loop() {
	// Send initial data on start:
	g.identify()
	ticker := time.NewTicker(g.period)
	for !g.stopSig.IsDone() {
		select {
		case <-ticker.C:
			go g.identify()
		case <-g.stopSig.Done():
			ticker.Stop()
			return
		}
	}
}

func (g *gatherer) Start(opts ...telemeter.Option) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopSig.IsDone() {
		g.reset()
		{
			g.gathering.Lock()
			g.opts = opts
			g.gathering.Unlock()
		}
		go g.loop()
	}
}

func (g *gatherer) Stop() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stopSig.Signal()
}

func (g *gatherer) AddGatherer(f GatherFunc) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gatherFuncs = append(g.gatherFuncs, f)
}

// AddTotal sets an entry in the props map with key and number returned by f as
// the value.
func AddTotal(ctx context.Context, props map[string]any, key string, f func(context.Context) (int, error)) error {
	ps, err := f(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s", key)
	}
	props["Total "+key] = ps
	return nil
}
