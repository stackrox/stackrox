package phonehome

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// GatherFunc returns properties gathered by a data source.
type GatherFunc func(context.Context) (map[string]any, error)

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
	AddGatherer(GatherFunc)
}

type nilGatherer struct{}

func (*nilGatherer) Start()                 {}
func (*nilGatherer) Stop()                  {}
func (*nilGatherer) AddGatherer(GatherFunc) {}

type gatherer struct {
	clientID    string
	telemeter   Telemeter
	period      time.Duration
	stopSig     concurrency.Signal
	ctx         context.Context
	mu          sync.Mutex
	gatherFuncs []GatherFunc
}

func newGatherer(clientID string, t Telemeter, p time.Duration) *gatherer {
	return &gatherer{
		clientID:  clientID,
		telemeter: t,
		period:    p,
	}
}

func (g *gatherer) reset() {
	g.stopSig.Reset()
	g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
}

func (g *gatherer) collect() map[string]any {
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

func (g *gatherer) loop() {
	ticker := time.NewTicker(g.period)
	for !g.stopSig.IsDone() {
		select {
		case <-ticker.C:
			go func() {
				g.telemeter.Identify(g.clientID, g.collect())
			}()
		case <-g.stopSig.Done():
			ticker.Stop()
			return
		}
	}
}

func (g *gatherer) Start() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopSig.IsDone() {
		g.reset()
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
