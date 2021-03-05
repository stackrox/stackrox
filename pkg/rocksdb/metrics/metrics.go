package metrics

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

const (
	// RocksDBDirName it the name of the RocksDB directory on the PVC
	RocksDBDirName = `rocksdb`
)

var (
	rocksDBPrefixSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "rocksdb_prefix_size",
		Help:      "RocksDB prefix size (equivalent to bolt bucket)",
	}, []string{"Prefix", "Type"})

	rocksDBPrefixBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "rocksdb_prefix_bytes",
		Help:      "RocksDB prefix bytes (equivalent to bolt bucket)",
	}, []string{"Prefix", "Type"})
)

func init() {
	prometheus.MustRegister(
		rocksDBPrefixSize,
		rocksDBPrefixBytes,
	)
}

// UpdateRocksDBPrefixSizeMetric sets the rocksdb metric for number of objects with a specific prefix
func UpdateRocksDBPrefixSizeMetric(db *rocksdb.RocksDB, prefix []byte, metricPrefix, objType string) {
	var count, bytes int
	err := generic.DefaultBucketForEach(db, prefix, false, func(k, v []byte) error {
		count++
		bytes += len(k) + len(v)
		return nil
	})
	if err != nil {
		return
	}
	rocksDBPrefixSize.With(prometheus.Labels{"Prefix": metricPrefix, "Type": objType}).Set(float64(count))
	rocksDBPrefixBytes.With(prometheus.Labels{"Prefix": metricPrefix, "Type": objType}).Set(float64(bytes))
}

// GetRocksDBMetrics returns a list of cardinality metrics per prefix and a list of size-in-bytes metrics per prefix
func GetRocksDBMetrics() ([]*dto.Metric, []*dto.Metric, error) {
	errList := errorhelpers.NewErrorList("errors collecting rocksdb metrics")
	cardinality, err := metrics.CollectToSlice(rocksDBPrefixSize)
	errList.AddError(errors.Wrap(err, "cardinality"))
	bytes, err := metrics.CollectToSlice(rocksDBPrefixBytes)
	errList.AddError(errors.Wrap(err, "bytes"))
	return cardinality, bytes, errList.ToError()
}

// GetRocksDBPath is the full directory path for rockdb in dbPath
func GetRocksDBPath(dbPath string) string {
	return filepath.Join(dbPath, RocksDBDirName)
}
