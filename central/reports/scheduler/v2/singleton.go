package v2

import (
	"github.com/prometheus/client_golang/prometheus"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	nodeReportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator/node"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	sched Scheduler
)

func initialize() {
	collectionDatastore, _ := collectionDS.Singleton()
	sched = New(
		reportConfigDS.Singleton(),
		reportSnapshotDS.Singleton(),
		collectionDatastore,
		notifierDS.Singleton(),
		reportGen.Singleton(),
		nodeReportGen.Singleton(),
		validation.Singleton(),
	)
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Name:      "report_scheduler_ready",
		Help:      "Whether this process owns report scheduling and has initialized recovery (1 ready, 0 inactive or waiting).",
	}, func() float64 {
		if sched.Ready() {
			return 1
		}
		return 0
	}))
}

// Singleton will return a singleton instance of the v2 report scheduler
func Singleton() Scheduler {
	once.Do(initialize)
	return sched
}
