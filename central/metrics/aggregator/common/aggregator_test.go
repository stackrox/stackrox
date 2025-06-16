package common

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

type testFinding struct {
	OneOrMore
	index int // the index of the data sample from the testData array.
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

func Test_makeAggregationKey(t *testing.T) {
	testMetric := map[Label]string{
		"Cluster":   "value",
		"IsFixable": "false",
	}
	getter := func(label Label) string {
		return testMetric[label]
	}

	t.Run("matching", func(t *testing.T) {
		key, labels := makeAggregationKey(
			map[Label]Expression{
				"Cluster":   {{"=", "*al*"}},
				"IsFixable": {},
			},
			getter,
			testLabelOrder)
		assert.Equal(t, aggregationKey("value|false"), key)
		assert.Equal(t, prometheus.Labels{
			"Cluster":   "value",
			"IsFixable": "false",
		}, labels)
	})

	t.Run("not matching", func(t *testing.T) {
		key, labels := makeAggregationKey(
			map[Label]Expression{
				"Cluster":   {{"=", "missing"}},
				"IsFixable": {},
			},
			getter, testLabelOrder)
		assert.Equal(t, aggregationKey(""), key)
		assert.Nil(t, labels)
	})
}

func Test_collectMatchingLabels(t *testing.T) {
	i := 0
	for range collectMatchingLabels(map[Label]Expression{
		"label1": {&Condition{"=", "value"}, &Condition{"=", "value"}},
		"label2": {&Condition{"=", "value"}, &Condition{"=", "value"}},
		"label3": {},
	}, func(l Label) string {
		i++
		return "value"
	}) {
		break
	}
	assert.Equal(t, 1, i)
}
