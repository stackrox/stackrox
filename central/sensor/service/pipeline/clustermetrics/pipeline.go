package clustermetrics

import (
	"context"

	clusterTelemetry "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/info"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
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
	return &pipelineImpl{metricsStore: &prometheusStore{}, infoMetric: info.Singleton()}
}

type usageStore interface {
	UpdateUsage(clusterID string, metrics *central.ClusterMetrics) error
}

// NewPipeline returns a new instance of the pipeline.
func NewPipeline(metricsStore MetricsStore, infoMetric info.Info, usageStore usageStore) pipeline.Fragment {
	return &pipelineImpl{metricsStore: metricsStore, infoMetric: infoMetric, usageStore: usageStore}
}

type pipelineImpl struct {
	pipeline.Fragment

	metricsStore MetricsStore
	infoMetric   info.Info
	usageStore   usageStore
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
	p.infoMetric.SetClusterMetrics(clusterID, clusterMetrics)
	_ = p.usageStore.UpdateUsage(clusterID, clusterMetrics)
	clusterTelemetry.UpdateSecuredClusterIdentity(ctx, clusterID, clusterMetrics)
	return nil
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.metricsStore.Set(clusterID, &central.ClusterMetrics{})
	p.infoMetric.DeleteClusterMetrics(clusterID)
}
