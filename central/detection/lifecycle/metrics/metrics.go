package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		processFilterCounter,
		matcherProcessIndicators,
	)
}

var (
	processFilterCounter = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_filter",
		Help:      "Process filter hits and misses",
	}, []string{"Type"})

	matcherProcessIndicators = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "matcher_process_indicators",
		Help:      "Number of process indicators filtered out by PlatformMatcher",
	})
)

// ProcessFilterCounterInc increments a counter for determining how effective the process filter is
func ProcessFilterCounterInc(typ string) {
	processFilterCounter.With(prometheus.Labels{"Type": typ}).Inc()
}

// MatcherProcessIndicatorsInc increments the counter for process indicators filtered out by PlatformMatcher.
func MatcherProcessIndicatorsInc() {
	matcherProcessIndicators.Inc()
}
