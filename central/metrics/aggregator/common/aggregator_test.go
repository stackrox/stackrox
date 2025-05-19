package common

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func Test_aggregator(t *testing.T) {
	a := makeAggregator(makeTestMetricLabelExpressions(t), testLabelOrder)
	assert.NotNil(t, a)
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {},
		"Test_aggregator_metric2": {},
	}, a.result)

	a.count(func(label Label) string { return testData[0][label] })

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  1,
			}},
		"Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  1,
			}},
	}, a.result)

	a.count(func(label Label) string { return testData[0][label] })

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  2,
			}},
		"Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  2,
			}},
	}, a.result)

	a.count(func(label Label) string { return testData[1][label] })

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[1]["Cluster"], "Severity": testData[1]["Severity"]},
				total:  1,
			},
		},
		"Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  2,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[1]["Namespace"]},
				total:  1,
			},
		},
	}, a.result)

	a.count(func(label Label) string { return testData[1][label] })

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[1]["Cluster"], "Severity": testData[1]["Severity"]},
				total:  2,
			},
		},
		"Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  2,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[1]["Namespace"]},
				total:  2,
			},
		},
	}, a.result)
}
