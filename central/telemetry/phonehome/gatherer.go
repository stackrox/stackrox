package phonehome

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	pkgPH "github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	log              = logging.LoggerForModule()
	gathererInstance *gatherer
	onceGatherer     sync.Once
)

// Time period for static data gathering from data sources.
const period = 5 * time.Minute

type gatherer struct {
	telemeter   pkgPH.Telemeter
	period      time.Duration
	stopSig     concurrency.Signal
	ctx         context.Context
	mu          sync.Mutex
	gatherFuncs []pkgPH.GatherFunc
}

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
	AddGatherer(pkgPH.GatherFunc)
}

func (g *gatherer) reset() {
	g.stopSig.Reset()
	g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
}

func newGatherer(t pkgPH.Telemeter, p time.Duration) *gatherer {
	return &gatherer{
		telemeter: t,
		period:    p,
	}
}

// GathererSingleton returns the telemetry gatherer instance.
func GathererSingleton() Gatherer {
	if pkgPH.Enabled() {
		onceGatherer.Do(func() {
			gathererInstance = newGatherer(pkgPH.TelemeterSingleton(), period)
		})
	}
	return gathererInstance
}

func (g *gatherer) collect() pkgPH.Properties {
	result := make(pkgPH.Properties)
	for i, f := range g.gatherFuncs {
		props, err := f(g.ctx)
		if err != nil {
			log.Errorf("gatherer %d failure: %v", i, err)
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
				if props := g.collect(); g.telemeter != nil {
					g.telemeter.Identify(props)
				}
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

func (g *gatherer) AddGatherer(f pkgPH.GatherFunc) {
	g.gatherFuncs = append(g.gatherFuncs, f)
}
