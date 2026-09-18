package lifecycle

import (
	"context"
	"time"

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
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/set"
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
	HandleNodeAlerts(clusterID string, alerts []*storage.Alert, stage storage.LifecycleStage) error
	DeploymentRemoved(deploymentID string) error
	RemovePolicy(policyID string) error
	RemoveDeploymentFromObservation(deploymentID string)
	SendBaselineToSensor(baseline *storage.ProcessBaseline) error
}

// newManager returns a new manager with the injected dependencies.
func newManager(buildTimeDetector buildtime.Detector, deployTimeDetector deploytime.Detector, runtimeDetector runtime.Detector,
	clusterDatastore clusterDatastore.DataStore, deploymentDatastore deploymentDatastore.DataStore, processesDataStore processDatastore.DataStore,
	baselines baselineDataStore.DataStore, alertManager alertmanager.AlertManager, reprocessor reprocessor.Loop,
	deletedDeploymentsCache cache.DeletedDeployments, filter filter.Filter, connectionManager connection.Manager) *managerImpl {
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

		deploymentObservationQueue: queue.New(),

		removedOrDisabledPolicies: set.NewStringSet(),

		connectionManager: connectionManager,
	}

	m.indicatorAccumulator = &backgroundworker.BatchAccumulator[*storage.ProcessIndicator]{
		Name:                "lifecycle-indicator-flush",
		FlushInterval:       indicatorFlushTickerDuration,
		EagerFlushRateLimit: rateLimitDuration,
		Collector: &backgroundworker.MapCollector[string, *storage.ProcessIndicator]{
			KeyFunc: func(indicator *storage.ProcessIndicator) string { return indicator.GetId() },
		},
		Flush: m.flushIndicatorQueue,
	}
	m.baselineFlushWorker = &backgroundworker.PeriodicWorker{
		Name:          "lifecycle-baseline-flush",
		Interval:      baselineFlushTickerDuration,
		SkipIfRunning: true,
		Run: func(_ context.Context) error {
			m.flushBaselineQueue()
			return nil
		},
	}

	backgroundworker.Global.Register(m.indicatorAccumulator)
	backgroundworker.Global.Register(m.baselineFlushWorker)

	m.indicatorAccumulator.Start(context.Background())
	m.baselineFlushWorker.Start(context.Background())

	return m
}
