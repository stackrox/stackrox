package clustermetrics

import (
	"context"

	clusterTelemetry "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	usageDataStore "github.com/stackrox/rox/central/usage/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

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
	return &pipelineImpl{metricsStore: &prometheusStore{}}
}

// NewPipeline returns a new instance of the pipeline.
func NewPipeline(metricsStore MetricsStore) pipeline.Fragment {
	return &pipelineImpl{metricsStore: metricsStore}
}

type pipelineImpl struct {
	pipeline.Fragment

	metricsStore MetricsStore
	usageStore   usageDataStore.DataStore
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
	p.metricsStore.Set(clusterID, msg.GetClusterMetrics())

	clusterTelemetry.UpdateSecuredClusterIdentity(ctx, clusterID, msg.GetClusterMetrics())

	p.usageStore.UpdateUsage(clusterID, msg.GetClusterMetrics())

	return nil
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.metricsStore.Set(clusterID, &central.ClusterMetrics{})
}
