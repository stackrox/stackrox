package globalindex

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(bleveDiskUsage)
}

const (
	diskUsageScrapeRate = 10 * time.Second

	metricPrefix = "bleve"
)

var (
	metricMap = make(map[string]prometheus.Gauge)

	bleveDiskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "bleve_disk_usage",
		Help:      "Amount of disk that Bleve is currently using",
	})
)

func newGauge(name string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help:      "The name should be descriptive enough",
	})
}

func walkStatsMap(parentPrefix string, m map[string]interface{}) {
	for k, v := range m {
		currPrefix := parentPrefix + "_" + k
		if subMap, ok := v.(map[string]interface{}); ok {
			walkStatsMap(currPrefix, subMap)
			continue
		}

		switch value := v.(type) {
		case uint64:
			gauge, ok := metricMap[currPrefix]
			if !ok {
				gauge = newGauge(currPrefix)
				metricMap[currPrefix] = gauge

				// Register the gauge the first time
				prometheus.MustRegister(gauge)
			}
			gauge.Set(float64(value))
		default:
			log.Warnf("Unhandled metric %q", currPrefix)
		}
	}
}

// start monitoring on that path
func startMonitoring(index bleve.Index, path string) {
	ticker := time.NewTicker(diskUsageScrapeRate)
	for range ticker.C {
		walkStatsMap(metricPrefix, index.StatsMap())
		size, err := fileutils.DirectorySize(path)
		if err != nil {
			log.Errorf("error getting index directory size: %v", err)
			continue
		}
		bleveDiskUsage.Set(float64(size))
	}
}
