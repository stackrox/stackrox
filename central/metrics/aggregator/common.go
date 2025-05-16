package aggregator

import "github.com/prometheus/client_golang/prometheus"

type Label string
type metricName string
type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true
type metricsConfig map[metricName]map[Label][]*expression

type record struct {
	labels prometheus.Labels
	total  int
}

type result map[metricName]map[metricKey]*record
