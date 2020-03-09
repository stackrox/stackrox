package lifecycle

import (
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"golang.org/x/time/rate"
)

const (
	rateLimitDuration            = 10 * time.Second
	indicatorFlushTickerDuration = 1 * time.Minute
)

var (
	log                = logging.LoggerForModule()
	alertsLockPoolSize = uint32(64) // Arbitrary choice.  Larger numbers decrease the likelihood unrelated things will try for the same lock.
)

// A Manager manages deployment/policy lifecycle updates.
//go:generate mockgen-wrapper
type Manager interface {
	IndicatorAdded(indicator *storage.ProcessIndicator) error
	UpsertPolicy(policy *storage.Policy) error
	HandleAlerts(deploymentID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	DeploymentRemoved(deployment *storage.Deployment) error
	RemovePolicy(policyID string) error
}

// newManager returns a new manager with the injected dependencies.
func newManager(enricher enrichment.Enricher, deploytimeDetector deploytime.Detector, runtimeDetector runtime.Detector,
	deploymentDatastore deploymentDatastore.DataStore, processesDataStore processDatastore.DataStore, whitelists whitelistDataStore.DataStore,
	imageDataStore imageDataStore.DataStore, alertManager alertmanager.AlertManager, riskManager riskManager.Manager,
	reprocessor reprocessor.Loop, deletedDeploymentsCache expiringcache.Cache, filter filter.Filter) *managerImpl {
	m := &managerImpl{
		enricher:                enricher,
		riskManager:             riskManager,
		deploytimeDetector:      deploytimeDetector,
		runtimeDetector:         runtimeDetector,
		alertManager:            alertManager,
		deploymentDataStore:     deploymentDatastore,
		processesDataStore:      processesDataStore,
		whitelists:              whitelists,
		imageDataStore:          imageDataStore,
		reprocessor:             reprocessor,
		deletedDeploymentsCache: deletedDeploymentsCache,
		processFilter:           filter,

		queuedIndicators: make(map[string]*storage.ProcessIndicator),

		indicatorRateLimiter: rate.NewLimiter(rate.Every(rateLimitDuration), 5),
		indicatorFlushTicker: time.NewTicker(indicatorFlushTickerDuration),

		policyAlertsLock: concurrency.NewKeyedMutex(alertsLockPoolSize),
	}

	go m.flushQueuePeriodically()
	return m
}
