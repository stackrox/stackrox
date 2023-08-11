package clustermetrics

import (
	"context"

	clusterTelemetry "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/telemetry"
	usageDS "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// MetricsStore persists a measurement of ClusterMetrics.
//
//go:generate mockgen-wrapper
type MetricsStore interface {
	Set(string, *central.ClusterMetrics)
}

type prometheusStore struct{}

func (prometheusStore) Set(clusterID string, cm *central.ClusterMetrics) {
	if cm != nil {
		metrics.SetClusterMetrics(clusterID, cm)
	}
}

// GetPipeline returns an instantiation of this particular pipeline.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(&prometheusStore{}, telemetry.Singleton(), usageDS.Singleton())
}

// NewPipeline returns a new instance of the pipeline.
func NewPipeline(metricsStore MetricsStore, telemetryMetrics telemetry.Telemetry, usageStore usageDS.DataStore) pipeline.Fragment {
	return &pipelineImpl{metricsStore: metricsStore, telemetryMetrics: telemetryMetrics, usageStore: usageStore}
}

type pipelineImpl struct {
	pipeline.Fragment

	metricsStore     MetricsStore
	telemetryMetrics telemetry.Telemetry
	usageStore       usageDS.DataStore
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetClusterMetrics() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(
	ctx context.Context,
	clusterID string,
	msg *central.MsgFromSensor,
	_ common.MessageInjector,
) error {
	clusterMetrics := msg.GetClusterMetrics()
	p.metricsStore.Set(clusterID, clusterMetrics)
	p.telemetryMetrics.SetClusterMetrics(clusterID, clusterMetrics)
	if err := p.usageStore.UpdateUsage(ctx, clusterID, &storage.SecuredUnits{
		NumNodes:    clusterMetrics.GetNodeCount(),
		NumCpuUnits: clusterMetrics.GetCpuCapacity(),
	}); err != nil {
		logging.GetRateLimitedLogger().Warn(
			"Error while trying to update secured units usage:", err.Error())
	}
	clusterTelemetry.UpdateSecuredClusterIdentity(ctx, clusterID, clusterMetrics)
	return nil
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.metricsStore.Set(clusterID, &central.ClusterMetrics{})
	p.telemetryMetrics.DeleteClusterMetrics(clusterID)
}
