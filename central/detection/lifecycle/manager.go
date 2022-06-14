package lifecycle

import (
	"time"

	"github.com/stackrox/stackrox/central/activecomponent/updater/aggregator"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/deployment/queue"
	"github.com/stackrox/stackrox/central/detection/alertmanager"
	"github.com/stackrox/stackrox/central/detection/deploytime"
	"github.com/stackrox/stackrox/central/detection/runtime"
	baselineDataStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/expiringcache"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/process/filter"
	"github.com/stackrox/stackrox/pkg/set"
	"golang.org/x/time/rate"
)

const (
	rateLimitDuration            = 10 * time.Second
	indicatorFlushTickerDuration = 1 * time.Minute
	baselineFlushTickerDuration  = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// A Manager manages deployment/policy lifecycle updates.
//go:generate mockgen-wrapper
type Manager interface {
	IndicatorAdded(indicator *storage.ProcessIndicator) error
	UpsertPolicy(policy *storage.Policy) error
	HandleDeploymentAlerts(deploymentID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	HandleResourceAlerts(clusterID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	DeploymentRemoved(deploymentID string) error
	RemovePolicy(policyID string) error
	RemoveDeploymentFromObservation(deploymentID string)
}

// newManager returns a new manager with the injected dependencies.
func newManager(deploytimeDetector deploytime.Detector, runtimeDetector runtime.Detector,
	deploymentDatastore deploymentDatastore.DataStore, processesDataStore processDatastore.DataStore, baselines baselineDataStore.DataStore,
	alertManager alertmanager.AlertManager, reprocessor reprocessor.Loop, deletedDeploymentsCache expiringcache.Cache, filter filter.Filter,
	processAggregator aggregator.ProcessAggregator) *managerImpl {
	m := &managerImpl{
		deploytimeDetector:      deploytimeDetector,
		runtimeDetector:         runtimeDetector,
		alertManager:            alertManager,
		deploymentDataStore:     deploymentDatastore,
		processesDataStore:      processesDataStore,
		baselines:               baselines,
		reprocessor:             reprocessor,
		deletedDeploymentsCache: deletedDeploymentsCache,
		processFilter:           filter,

		queuedIndicators:           make(map[string]*storage.ProcessIndicator),
		deploymentObservationQueue: queue.New(),

		indicatorRateLimiter: rate.NewLimiter(rate.Every(rateLimitDuration), 5),
		indicatorFlushTicker: time.NewTicker(indicatorFlushTickerDuration),
		baselineFlushTicker:  time.NewTicker(baselineFlushTickerDuration),

		removedOrDisabledPolicies: set.NewStringSet(),
		processAggregator:         processAggregator,
	}

	go m.flushQueuePeriodically()
	go m.flushBaselineQueuePeriodically()
	return m
}
