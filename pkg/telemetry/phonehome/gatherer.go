package phonehome

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	onceGatherer sync.Once
)

type gatherer struct {
	clientID    string
	telemeter   Telemeter
	period      time.Duration
	stopSig     concurrency.Signal
	ctx         context.Context
	mu          sync.Mutex
	gatherFuncs []GatherFunc
}

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
	AddGatherer(GatherFunc)
}

func (g *gatherer) reset() {
	g.stopSig.Reset()
	g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
}

func newGatherer(clientID string, t Telemeter, p time.Duration) *gatherer {
	return &gatherer{
		clientID:  clientID,
		telemeter: t,
		period:    p,
	}
}

// Gatherer returns the telemetry gatherer instance.
func (cfg *Config) Gatherer() Gatherer {
	onceGatherer.Do(func() {
		if cfg.Enabled() {
			period := cfg.GatherPeriod
			if cfg.GatherPeriod.Nanoseconds() == 0 {
				period = 1 * time.Hour
			}
			cfg.gatherer = newGatherer(cfg.ClientID, cfg.Telemeter(), period)
		} else {
			cfg.gatherer = (*gatherer)(nil)
		}
	})
	return cfg.gatherer
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
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gatherFuncs = append(g.gatherFuncs, f)
}
