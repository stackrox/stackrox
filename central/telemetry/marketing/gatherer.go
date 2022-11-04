package marketing

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
	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

type marketing struct {
	telemeter mpkg.Telemeter
	period    time.Duration
	ticker    *time.Ticker
	stopSig   concurrency.Signal
	ctx       context.Context
	cancel    context.CancelFunc
	userAgent string
}

var (
	m    *marketing
	once sync.Once
)

// InitGatherer initializes the periodic telemetry data gatherer.
func InitGatherer(t mpkg.Telemeter, p time.Duration) {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		m = &marketing{
			telemeter: t,
			period:    p,
			userAgent: "central/" + version.GetMainVersion(),
			ctx:       sac.WithAllAccess(ctx),
			cancel:    cancel,
			stopSig:   concurrency.NewSignal(),
		}
	})
}

// Singleton returns the previously initialized telemeter instance.
func Singleton() *marketing {
	return m
}

func (m *marketing) loop() {
	for !m.stopSig.IsDone() {
		select {
		case <-m.ticker.C:
			go m.gather()
		case <-m.stopSig.Done():
			return
		}
	}
	log.Debug("Loop stopped.")
}

func (m *marketing) Start() {
	if mpkg.Enabled() {
		m.telemeter.Start()
		m.ticker = time.NewTicker(m.period)
		go m.loop()
		log.Debug("Marketing telemetry data collection ticker enabled.")
	}
}

func (m *marketing) Stop() {
	if m != nil {
		m.telemeter.Stop()
		m.cancel()
		m.stopSig.Signal()
	}
}

func addTotal[T any](props map[string]any, key string, f func(context.Context) ([]*T, error)) {
	ps, err := f(Singleton().ctx)
	if err != nil {
		log.Errorf("Failed to get %s: %v", key, err)
	} else {
		props["Total "+key] = len(ps)
	}
}

func (m *marketing) gather() {
	log.Debug("Starting marketing telemetry data collection.")
	defer log.Debug("Done with marketing telemetry data collection.")

	totals := make(map[string]any)
	rs := roles.Singleton()

	addTotal(totals, "PermissionSets", rs.GetAllPermissionSets)
	addTotal(totals, "Roles", rs.GetAllRoles)
	addTotal(totals, "Access Scopes", rs.GetAllAccessScopes)
	addTotal(totals, "Signature Integrations", si.Singleton().GetAllSignatureIntegrations)

	groups, err := groupDataStore.Singleton().GetAll(m.ctx)
	if err != nil {
		log.Error("Failed to get Groups: ", err)
		return
	}
	providers, err := apDataStore.Singleton().GetAllAuthProviders(m.ctx)
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
	m.telemeter.Identify(totals)
}
