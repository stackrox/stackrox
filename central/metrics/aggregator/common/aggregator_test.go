package common

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

type testFinding struct {
	OneOrMore
	index int
}

func Test_aggregator(t *testing.T) {
	getters := map[Label]func(testFinding) string{
		"Severity":  func(tf testFinding) string { return testData[tf.index]["Severity"] },
		"Cluster":   func(tf testFinding) string { return testData[tf.index]["Cluster"] },
		"Namespace": func(tf testFinding) string { return testData[tf.index]["Namespace"] },
	}
	a := makeAggregator(makeTestMetricLabelExpression(t), testLabelOrder, getters)
	assert.NotNil(t, a)
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {},
		"Test_aggregator_metric2": {},
	}, a.result)

	a.count(testFinding{index: 0})

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

	a.count(testFinding{index: 0})

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

	a.count(testFinding{index: 1})

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

	a.count(testFinding{OneOrMore: 5, index: 1})

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[1]["Cluster"], "Severity": testData[1]["Severity"]},
				total:  6,
			},
		},
		"Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  2,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[1]["Namespace"]},
				total:  6,
			},
		},
	}, a.result)
}
