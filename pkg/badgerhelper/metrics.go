package badgerhelper

import (
	"github.com/dgraph-io/badger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		badgerPrefixSize,
	)
}

var (
	badgerPrefixSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "badger_prefix_size",
		Help:      "Badger prefix size (equivalent to bolt bucket)",
	}, []string{"Prefix", "Type"})
)

// UpdateBadgerPrefixSizeMetric sets the badger metric for number of objects with a specific prefix
func UpdateBadgerPrefixSizeMetric(db *badger.DB, prefix, objType string) {
	var count int
	err := db.View(func(txn *badger.Txn) error {
		count = Count(txn, []byte(prefix))
		return nil
	})
	if err != nil {
		return
	}
	badgerPrefixSize.With(prometheus.Labels{"Prefix": prefix, "Type": objType}).Set(float64(count))
}
