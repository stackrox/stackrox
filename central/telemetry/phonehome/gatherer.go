package phonehome

import (
	"context"
	"sync"
	"time"

	apDataStore "github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roles "github.com/stackrox/rox/central/role/datastore"
	si "github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgPH "github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
	g   *gatherer
)

const period = 5 * time.Minute

type gatherer struct {
	telemeter pkgPH.Telemeter
	period    time.Duration
	ticker    *time.Ticker
	stopSig   concurrency.Signal
	ctx       context.Context
	cancel    context.CancelFunc
	userAgent string
	mu        sync.Mutex
}

// Gatherer interface for interacting with telemetry gatherer.
type Gatherer interface {
	Start()
	Stop()
}

// GathererSingleton returns the telemetry gatherer instance.
func GathererSingleton() Gatherer {
	if Enabled() {
		once.Do(func() {
			ctx, cancel := context.WithCancel(context.Background())
			g = &gatherer{
				telemeter: TelemeterSingleton(),
				period:    period,
				userAgent: "central/" + version.GetMainVersion(),
				ctx:       sac.WithAllAccess(ctx),
				cancel:    cancel,
				stopSig:   concurrency.NewSignal(),
			}
		})
	}
	return g
}

func (g *gatherer) loop() {
	for !g.stopSig.IsDone() {
		select {
		case <-g.ticker.C:
			go g.gather()
		case <-g.stopSig.Done():
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
	if g.ticker == nil {
		g.ticker = time.NewTicker(g.period)
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
	g.ticker = nil
}

func addTotal[T any](props map[string]any, key string, f func(context.Context) ([]*T, error)) {
	if ps, err := f(g.ctx); err != nil {
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

	addTotal(totals, "PermissionSets", rs.GetAllPermissionSets)
	addTotal(totals, "Roles", rs.GetAllRoles)
	addTotal(totals, "Access Scopes", rs.GetAllAccessScopes)
	addTotal(totals, "Signature Integrations", si.Singleton().GetAllSignatureIntegrations)

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
