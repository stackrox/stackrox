package lifecycle

import (
	"time"

	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/deployment/cache"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/queue"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/runtime"
	baselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/set"
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
//
//go:generate mockgen-wrapper
type Manager interface {
	IndicatorAdded(indicator *storage.ProcessIndicator) error
	UpsertPolicy(policy *storage.Policy) error
	HandleDeploymentAlerts(deploymentID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	HandleResourceAlerts(clusterID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	DeploymentRemoved(deploymentID string) error
	RemovePolicy(policyID string) error
	RemoveDeploymentFromObservation(deploymentID string)
	SendBaselineToSensor(baseline *storage.ProcessBaseline) error
}

// newManager returns a new manager with the injected dependencies.
func newManager(buildTimeDetector buildtime.Detector, deployTimeDetector deploytime.Detector, runtimeDetector runtime.Detector,
	clusterDatastore clusterDatastore.DataStore, deploymentDatastore deploymentDatastore.DataStore, processesDataStore processDatastore.DataStore,
	baselines baselineDataStore.DataStore, alertManager alertmanager.AlertManager, reprocessor reprocessor.Loop,
	deletedDeploymentsCache cache.DeletedDeployments, filter filter.Filter, processAggregator aggregator.ProcessAggregator,
	connectionManager connection.Manager) *managerImpl {
	m := &managerImpl{
		buildTimeDetector:       buildTimeDetector,
		deployTimeDetector:      deployTimeDetector,
		runtimeDetector:         runtimeDetector,
		alertManager:            alertManager,
		clusterDataStore:        clusterDatastore,
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

		connectionManager: connectionManager,
	}

	go m.flushQueuePeriodically()
	go m.flushBaselineQueuePeriodically()
	return m
}
