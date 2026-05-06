package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		processFilterCounter,
		processIndicatorsNotPersisted,
	)
}

var (
	processFilterCounter = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_filter",
		Help:      "Process filter hits and misses",
	}, []string{"Type"})

	processIndicatorsNotPersisted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_not_persisted",
		Help:      "Number of process indicators filtered out and not persisted",
	})
)

// ProcessFilterCounterInc increments a counter for determining how effective the process filter is
func ProcessFilterCounterInc(typ string) {
	processFilterCounter.With(prometheus.Labels{"Type": typ}).Inc()
}

// ProcessIndicatorsNotPersistedInc increments the counter for process indicators filtered out and not persisted.
func ProcessIndicatorsNotPersistedInc() {
	processIndicatorsNotPersisted.Inc()
}
