package info

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once sync.Once
	info *infoImpl
)

// Info defines a metric series with meta data about Central.
//
//go:generate mockgen-wrapper
type Info interface {
	Start()
	SetClusterMetrics(string, *central.ClusterMetrics)
	DeleteClusterMetrics(string)
}

func initialize() {
	info = &infoImpl{cache: newMetricsCache(), gauge: newGaugeVec()}
	prometheus.MustRegister(info.gauge)
}

// Singleton provides the interface for setting the info metrics.
func Singleton() Info {
	once.Do(initialize)
	return info
}

type infoImpl struct {
	cache *metricsCache
	gauge *prometheus.GaugeVec
}

// Start registers the info metric.
func (i *infoImpl) Start() {
	if i == nil {
		return
	}
	i.gauge.WithLabelValues("0", "0", "0").Set(1)
}

// SetClusterMetrics updates the info metric with the cluster metrics.
func (i *infoImpl) SetClusterMetrics(clusterID string, clusterMetrics *central.ClusterMetrics) {
	if i == nil {
		return
	}
	i.cache.Set(clusterID, clusterMetrics)
	i.updateGauge()
}

// DeleteClusterMetrics deletes metrics with the cluster ID from the info metric.
func (i *infoImpl) DeleteClusterMetrics(clusterID string) {
	if i == nil {
		return
	}
	i.cache.Delete(clusterID)
	i.updateGauge()
}

func (i *infoImpl) updateGauge() {
	i.gauge.Reset()
	clusterMetricTotal := i.cache.Sum()
	i.gauge.WithLabelValues(
		strconv.Itoa(i.cache.Len()),
		strconv.FormatInt(clusterMetricTotal.GetNodeCount(), 10),
		strconv.FormatInt(clusterMetricTotal.GetCpuCapacity(), 10),
	).Set(1)
}
