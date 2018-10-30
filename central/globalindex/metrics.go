package globalindex

import (
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.Register(bleveDiskUsage)
}

const (
	diskUsageScrapeRate = 10 * time.Second
)

var (
	bleveDiskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "bleve_disk_usage",
		Help:      "Amount of disk that Bleve is currently using",
	})
)

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
		logger.Error(err)
	}
	return size
}

// start monitoring on that path
func startMonitoring(mossPath string) {
	ticker := time.NewTicker(diskUsageScrapeRate)
	for range ticker.C {
		bleveDiskUsage.Set(float64(directorySize(mossPath)))
	}
}
