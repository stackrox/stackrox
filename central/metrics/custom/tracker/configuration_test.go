package tracker

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

var testData = []map[Label]string{
	{
		"Severity":  "CRITICAL",
		"Cluster":   "cluster 1",
		"Namespace": "ns 1",
	}, {
		"Severity":  "HIGH",
		"Cluster":   "cluster 2",
		"Namespace": "ns 2",
	},
	{
		"Severity":  "LOW",
		"Cluster":   "cluster 3",
		"Namespace": "ns 3",
	},
	{
		"Severity":  "CRITICAL",
		"Cluster":   "cluster 1",
		"Namespace": "ns 3",
	},
	{
		"Severity":  "LOW",
		"Cluster":   "cluster 5",
		"Namespace": "ns 3",
	},
}

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetrics_MetricGroup_Labels {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetrics_MetricGroup_Labels{
		pfx + "_metric1": {Labels: []string{"Severity", "Cluster"}},
		pfx + "_metric2": {Labels: []string{"Namespace"}},
	}
}

func makeTestMetricConfiguration(t *testing.T) MetricsConfiguration {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricsConfiguration{
		pfx + "_metric1": {"Severity", "Cluster"},
		pfx + "_metric2": {"Namespace"},
	}
}
