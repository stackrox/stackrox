package globalindex

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

func init() {
	prometheus.MustRegister(bleveDiskUsage)
}

const (
	diskUsageScrapeRate = 10 * time.Second

	metricPrefix = "bleve"
)

var (
	metricMap     = make(map[string]*prometheus.GaugeVec)
	metricMapLock sync.Mutex

	bleveDiskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "bleve_disk_usage",
		Help:      "Amount of disk that Bleve is currently using",
	})
)

func newGauge(name string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help:      "The name should be descriptive enough",
	}, []string{"Index"})
}

func walkStatsMap(indexName, parentPrefix string, m map[string]interface{}) {
	for k, v := range m {
		currPrefix := parentPrefix + "_" + k
		switch value := v.(type) {
		case map[string]interface{}:
			walkStatsMap(indexName, currPrefix, value)
		case uint64:
			concurrency.WithLock(&metricMapLock, func() {
				gauge, ok := metricMap[currPrefix]
				if !ok {
					gauge = newGauge(currPrefix)
					metricMap[currPrefix] = gauge

					// Register the gauge the first time
					prometheus.MustRegister(gauge)
				}
				gauge.With(prometheus.Labels{"Index": indexName}).Set(float64(value))
			})
		default:
			log.Warnf("Unhandled metric %q", currPrefix)
		}
	}
}

// start monitoring on that path
func startMonitoring(index bleve.Index, resource string, path string) {
	ticker := time.NewTicker(diskUsageScrapeRate)
	for range ticker.C {
		walkStatsMap(resource, metricPrefix, index.StatsMap())
		size, err := fileutils.DirectorySize(path)
		if err != nil {
			log.Errorf("error getting index directory size: %v", err)
			continue
		}
		bleveDiskUsage.Set(float64(size))
	}
}
