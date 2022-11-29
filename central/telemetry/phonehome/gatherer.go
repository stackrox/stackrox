package phonehome

import (
	"context"
	"time"

	apDataStore "github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roles "github.com/stackrox/rox/central/role/datastore"
	si "github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	pkgPH "github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	log              = logging.LoggerForModule()
	gathererInstance *gatherer
	onceGatherer     sync.Once
)

const period = 5 * time.Minute

type gatherer struct {
	telemeter pkgPH.Telemeter
	period    time.Duration
	ticker    *time.Ticker
	stopSig   concurrency.Signal
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	f         func(*gatherer)
}

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
}

func (g *gatherer) reset() {
	ctx, cancel := context.WithCancel(context.Background())
	g.ctx = sac.WithAllAccess(ctx)
	g.cancel = cancel
	g.stopSig = concurrency.NewSignal()
}

func newGatherer(t pkgPH.Telemeter, p time.Duration, f func(*gatherer)) *gatherer {
	g := &gatherer{
		telemeter: t,
		period:    p,
		f:         f,
	}
	g.reset()
	return g
}

// GathererSingleton returns the telemetry gatherer instance.
func GathererSingleton() Gatherer {
	if Enabled() {
		onceGatherer.Do(func() {
			gathererInstance = newGatherer(TelemeterSingleton(), period, func(g *gatherer) { g.gather() })
		})
	}
	return gathererInstance
}

func (g *gatherer) loop() {
	g.ticker = time.NewTicker(g.period)
	for !g.stopSig.IsDone() {
		select {
		case <-g.ticker.C:
			go g.f(g)
		case <-g.stopSig.Done():
			g.ticker.Stop()
			g.ticker = nil
			g.cancel()
			return
		}
	}
	log.Debug("Loop stopped.")
}

func (g *gatherer) Start() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	// Ignore Start if the ticker is active.
	if g.ticker == nil {
		g.reset()
		go g.loop()
		log.Debug("Telemetry data collection ticker enabled.")
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

func addTotal[T any](ctx context.Context, props map[string]any, key string, f func(context.Context) ([]*T, error)) {
	if ps, err := f(ctx); err != nil {
		log.Errorf("Failed to get %s: %v", key, err)
	} else {
		props["Total "+key] = len(ps)
	}
}

func (g *gatherer) gather() {
	log.Debug("Starting telemetry data collection.")
	defer log.Debug("Done with telemetry data collection.")

	totals := make(map[string]any)
	rs := roles.Singleton()

	addTotal(g.ctx, totals, "PermissionSets", rs.GetAllPermissionSets)
	addTotal(g.ctx, totals, "Roles", rs.GetAllRoles)
	addTotal(g.ctx, totals, "Access Scopes", rs.GetAllAccessScopes)
	addTotal(g.ctx, totals, "Signature Integrations", si.Singleton().GetAllSignatureIntegrations)

	groups, err := groupDataStore.Singleton().GetAll(g.ctx)
	if err != nil {
		log.Error("Failed to get Groups: ", err)
		return
	}
	providers, err := apDataStore.Singleton().GetAllAuthProviders(g.ctx)
	if err != nil {
		log.Error("Failed to get AuthProviders: ", err)
		return
	}

	providerIDNames := make(map[string]string)
	providerNames := make([]string, len(providers))
	for _, provider := range providers {
		providerIDNames[provider.GetId()] = provider.GetName()
		providerNames = append(providerNames, provider.GetName())
	}
	totals["Auth Providers"] = providerNames

	providerGroups := make(map[string]int)
	for _, group := range groups {
		id := group.GetProps().GetAuthProviderId()
		providerGroups[id] = providerGroups[id] + 1
	}

	for id, n := range providerGroups {
		totals["Total Groups of "+providerIDNames[id]] = n
	}
	g.telemeter.Identify(totals)
}
