package clustermetrics

import (
	"context"

	"github.com/gogo/protobuf/types"
	clusterTelemetry "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/info"
	usageDS "github.com/stackrox/rox/central/productusage/datastore/securedunits"
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
func GetPipeline(usageStore usageDS.DataStore) pipeline.Fragment {
	return &pipelineImpl{metricsStore: &prometheusStore{}, infoMetric: info.Singleton(), usageStore: usageStore}
}

// NewPipeline returns a new instance of the pipeline.
func NewPipeline(metricsStore MetricsStore, infoMetric info.Info, usageStore usageDS.DataStore) pipeline.Fragment {
	return &pipelineImpl{metricsStore: metricsStore, infoMetric: infoMetric, usageStore: usageStore}
}

type pipelineImpl struct {
	pipeline.Fragment

	metricsStore MetricsStore
	infoMetric   info.Info
	usageStore   usageDS.DataStore
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetClusterMetrics() != nil
}

type usageData central.ClusterMetrics

func (*usageData) GetTimestamp() *types.Timestamp {
	return types.TimestampNow()
}

func (us *usageData) GetNumNodes() int64 {
	return (*central.ClusterMetrics)(us).GetNodeCount()
}

func (us *usageData) GetNumCPUUnits() int64 {
	return (*central.ClusterMetrics)(us).GetCpuCapacity()
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
	_ = p.usageStore.UpdateUsage(ctx, clusterID, (*usageData)(clusterMetrics))
	clusterTelemetry.UpdateSecuredClusterIdentity(ctx, clusterID, clusterMetrics)

	clusterTelemetry.UpdateSecuredClusterIdentity(ctx, clusterID, msg.GetClusterMetrics())

	return nil
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.metricsStore.Set(clusterID, &central.ClusterMetrics{})
	p.infoMetric.DeleteClusterMetrics(clusterID)
}
