package globalindex

import (
	"os"
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/prometheus/client_golang/prometheus"
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

func directorySize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return err
	})
	if err != nil {
		log.Error(err)
	}
	return size
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
		bleveDiskUsage.Set(float64(directorySize(path)))
		walkStatsMap(metricPrefix, index.StatsMap())
	}
}
