package clustermetrics

import (
	"context"

	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/logging"
)

var log = logging.LoggerForModule()

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// MetricsStore persists a measurement of ClusterMetrics.
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
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
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
	return nil
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.metricsStore.Set(clusterID, &central.ClusterMetrics{})
}
