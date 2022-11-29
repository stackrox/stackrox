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
	telemeter  pkgPH.Telemeter
	period     time.Duration
	stopSig    concurrency.Signal
	ctx        context.Context
	mu         sync.Mutex
	gatherFunc func(context.Context) (map[string]any, error)
}

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
}

func (g *gatherer) reset() {
	g.stopSig.Reset()
	g.ctx, _ = concurrency.DependentContext(context.Background(), &g.stopSig)
}

func newGatherer(t pkgPH.Telemeter, p time.Duration, f func(context.Context) (map[string]any, error)) *gatherer {
	return &gatherer{
		telemeter:  t,
		period:     p,
		gatherFunc: f,
	}
}

// GathererSingleton returns the telemetry gatherer instance.
func GathererSingleton() Gatherer {
	if Enabled() {
		onceGatherer.Do(func() {
			gathererInstance = newGatherer(TelemeterSingleton(), period, gather)
		})
	}
	return gathererInstance
}

func (g *gatherer) loop() {
	ticker := time.NewTicker(g.period)
	for !g.stopSig.IsDone() {
		select {
		case <-ticker.C:
			go func() {
				if props, err := g.gatherFunc(g.ctx); err == nil && g.telemeter != nil {
					g.telemeter.Identify(props)
				} else {
					log.Error(err)
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
