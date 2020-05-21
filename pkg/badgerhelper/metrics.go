package badgerhelper

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		badgerPrefixSize,
		badgerPrefixBytes,
	)
}

var (
	badgerPrefixSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "badger_prefix_size",
		Help:      "Badger prefix size (equivalent to bolt bucket)",
	}, []string{"Prefix", "Type"})
	badgerPrefixBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "badger_prefix_bytes",
		Help:      "Badger prefix bytes (equivalent to bolt bucket)",
	}, []string{"Prefix", "Type"})
)

// UpdateBadgerPrefixSizeMetric sets the badger metric for number of objects with a specific prefix
func UpdateBadgerPrefixSizeMetric(db *badger.DB, prefix []byte, metricPrefix, objType string) {
	var (
		count int
		bytes int
	)
	err := db.View(func(txn *badger.Txn) error {
		var err error
		count, bytes, err = CountWithBytes(txn, dbhelper.GetBucketKey(prefix, nil))
		return err
	})
	if err != nil {
		return
	}
	badgerPrefixSize.With(prometheus.Labels{"Prefix": metricPrefix, "Type": objType}).Set(float64(count))
	badgerPrefixBytes.With(prometheus.Labels{"Prefix": metricPrefix, "Type": objType}).Set(float64(bytes))
}

// GetBadgerMetrics returns a list of cardinality metrics per prefix and a list of size-in-bytes metrics per prefix
func GetBadgerMetrics() ([]*dto.Metric, []*dto.Metric, error) {
	errList := errorhelpers.NewErrorList("errors collecting badger metrics")
	cardinality, err := metrics.CollectToSlice(badgerPrefixSize)
	errList.AddError(errors.Wrap(err, "cardinality"))
	bytes, err := metrics.CollectToSlice(badgerPrefixBytes)
	errList.AddError(errors.Wrap(err, "bytes"))
	return cardinality, bytes, errList.ToError()
}
