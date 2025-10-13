package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	installationStore "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	telemetry *telemetryImpl
)

// Telemetry defines metric series with meta data about Central.
//
//go:generate mockgen-wrapper
type Telemetry interface {
	Start()
	SetClusterMetrics(string, *central.ClusterMetrics)
	DeleteClusterMetrics(string)
}

func initialize() {
	telemetry = &telemetryImpl{
		cache:  newMetricsCache(),
		gauges: newGaugeMap(installationStore.Singleton()),
	}
	for _, gauge := range telemetry.gauges {
		prometheus.MustRegister(gauge)
	}
}

// Singleton provides the interface for setting the telemetry metrics.
func Singleton() Telemetry {
	once.Do(initialize)
	return telemetry
}

type telemetryImpl struct {
	cache  *metricsCache
	gauges map[string]prometheus.Gauge
}

// Start registers the telemetry metric.
func (i *telemetryImpl) Start() {
	if i == nil {
		return
	}
	for _, gauge := range telemetry.gauges {
		gauge.Set(0)
	}
	i.gauges[infoGaugeName].Set(1)
}

// SetClusterMetrics updates the telemetry metric with the cluster metrics.
func (i *telemetryImpl) SetClusterMetrics(clusterID string, clusterMetrics *central.ClusterMetrics) {
	if i == nil {
		return
	}
	i.cache.Set(clusterID, clusterMetrics)
	i.updateGauges()
}

// DeleteClusterMetrics deletes metrics with the cluster ID from the telemetry metric.
func (i *telemetryImpl) DeleteClusterMetrics(clusterID string) {
	if i == nil {
		return
	}
	i.cache.Delete(clusterID)
	i.updateGauges()
}

func (i *telemetryImpl) updateGauges() {
	clusterMetricTotal := i.cache.Sum()
	i.gauges[securedClustersGaugeName].Set(float64(i.cache.Len()))
	i.gauges[securedNodesGaugeName].Set(float64(clusterMetricTotal.GetNodeCount()))
	i.gauges[securedVCPUGaugeName].Set(float64(clusterMetricTotal.GetCpuCapacity()))
}
