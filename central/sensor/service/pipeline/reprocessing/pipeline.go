package reprocessing

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/utils"
	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/reprocessor"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"golang.org/x/sync/semaphore"
)


var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)

	riskSemaphoreQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "central",
		Name:      "deployment_risk_semaphore_queue_size",
		Help:      "Number of deployment risk reprocessing operations waiting for a semaphore slot.",
	})
	riskSemaphoreHoldingSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "central",
		Name:      "deployment_risk_semaphore_holding_size",
		Help:      "Number of deployment risk reprocessing operations currently holding a semaphore slot.",
	})
	riskSemaphoreTimeouts = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "central",
		Name:      "deployment_risk_semaphore_timeouts_total",
		Help:      "Total number of deployment risk reprocessing operations that timed out waiting for a semaphore slot.",
	})
)

func init() {
	prometheus.MustRegister(riskSemaphoreQueueSize, riskSemaphoreHoldingSize, riskSemaphoreTimeouts)
}

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton(), lifecycle.SingletonManager(), riskManager.Singleton(), reprocessor.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(deployments datastore.DataStore, manager lifecycle.Manager, riskManager riskManager.Manager, riskReprocessor reprocessor.Loop) pipeline.Fragment {
	maxConcurrency := int64(env.DeploymentRiskMaxConcurrency.IntegerSetting())
	return &pipelineImpl{
		riskManager:     riskManager,
		riskReprocessor: riskReprocessor,
		manager:         manager,
		deployments:     deployments,
		riskSemaphore:   semaphore.NewWeighted(maxConcurrency),
	}
}

type pipelineImpl struct {
	deployments     datastore.DataStore
	riskManager     riskManager.Manager
	riskReprocessor reprocessor.Loop
	manager         lifecycle.Manager
	riskSemaphore   *semaphore.Weighted
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, _ *reconciliation.StoreMap) error {
	// Run reprocessing once sync has completed
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.deployments.Search(ctx, query)
	if err != nil {
		return err
	}
	s.riskReprocessor.ReprocessRiskForDeployments(search.ResultsToIDs(results)...)
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetReprocessDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.DeploymentReprocess)

	// Throttle concurrent risk reprocessing to prevent DB connection pool exhaustion.
	// Without this, 17 worker goroutines per cluster can all call ReprocessDeploymentRisk
	// simultaneously, each making ~10 DB operations (GetDeployment, GetRisk, image risks,
	// process indicators iteration, baseline evaluation, upserts). With multiple clusters
	// this easily exceeds the connection pool limit (default 90).
	//
	// The semaphore is acquired before GetDeployment to also gate the initial DB fetch,
	// preventing blocked goroutines from holding deserialized deployments in memory.
	//
	// Backpressure model: worker goroutines (17 per cluster) block here until a slot opens.
	// The upstream DedupingQueue deduplicates by deployment ID, so queue growth is bounded
	// by the number of unique deployments. Workers that are blocked here simply stop pulling
	// from their queue, which is the desired backpressure behavior.
	//
	// A timeout is applied to prevent indefinite blocking if risk operations are stuck.
	// On timeout, the operation is dropped -- it will be retried on the next reprocessing
	// cycle (default every 10 minutes via ROX_RISK_REPROCESSING_INTERVAL).
	if err := s.acquireRiskSemaphore(ctx); err != nil {
		return err
	}
	defer s.releaseRiskSemaphore()

	reprocessMsg := msg.GetEvent().GetReprocessDeployment()

	deployment, exists, err := s.deployments.GetDeployment(ctx, reprocessMsg.GetDeploymentId())
	if err != nil || !exists {
		return err
	}

	if features.FlattenImageData.Enabled() {
		// IDV2s may not be set if sensor is running an older version
		utils.PopulateContainerImageIDV2s(deployment)
	}

	s.riskManager.ReprocessDeploymentRisk(deployment)
	return nil
}

// acquireRiskSemaphore acquires the risk reprocessing semaphore with an optional timeout.
// This follows the same pattern as the image scan semaphore in central/image/service.
func (s *pipelineImpl) acquireRiskSemaphore(ctx context.Context) error {
	log.Infof("SHREWS -- Acquiring risk semaphore")
	waitTime := env.DeploymentRiskSemaphoreWaitTime.DurationSetting()

	acquireCtx := ctx
	if waitTime > 0 {
		var cancel context.CancelFunc
		acquireCtx, cancel = context.WithTimeout(ctx, waitTime)
		defer cancel()
	}

	riskSemaphoreQueueSize.Inc()
	defer riskSemaphoreQueueSize.Dec()

	if err := s.riskSemaphore.Acquire(acquireCtx, 1); err != nil {
		if ctx.Err() != nil {
			// Parent context was cancelled (sensor disconnected). This is expected.
			log.Debugf("Context cancelled while waiting to reprocess deployment risk: %v", err)
		} else {
			// Semaphore wait timed out. The deployment will be reprocessed on the next cycle.
			riskSemaphoreTimeouts.Inc()
			log.Warnf("Timed out waiting to reprocess deployment risk (waited %v, queue is full): %v",
				waitTime, err)
		}
		return err
	}

	log.Infof("SHREWS -- Acquired risk semaphore")
	riskSemaphoreHoldingSize.Inc()
	return nil
}

func (s *pipelineImpl) releaseRiskSemaphore() {
	log.Infof("SHREWS -- Releasing risk semaphore")
	s.riskSemaphore.Release(1)
	riskSemaphoreHoldingSize.Dec()
}

func (s *pipelineImpl) OnFinish(_ string) {}
