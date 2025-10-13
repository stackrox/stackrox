package tracker

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func Test_aggregator(t *testing.T) {
	getters := map[Label]func(testFinding) string{
		"Severity":  func(tf testFinding) string { return testData[tf]["Severity"] },
		"Cluster":   func(tf testFinding) string { return testData[tf]["Cluster"] },
		"Namespace": func(tf testFinding) string { return testData[tf]["Namespace"] },
	}
	a := makeAggregator(makeTestMetricDescriptors(t), testLabelOrder, getters)
	assert.NotNil(t, a)
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"test_Test_aggregator_metric1": {},
		"test_Test_aggregator_metric2": {},
	}, a.result)

	a.count(testFinding(0))

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"test_Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  1,
			}},
		"test_Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  1,
			}},
	}, a.result)

	a.count(testFinding(0))

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"test_Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{
					"Cluster":  testData[0]["Cluster"],
					"Severity": testData[0]["Severity"]},
				total: 2,
			}},
		"test_Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{
					"Namespace": testData[0]["Namespace"]},
				total: 2,
			}},
	}, a.result)

	a.count(testFinding(1))

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"test_Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{
					"Cluster":  testData[0]["Cluster"],
					"Severity": testData[0]["Severity"]},
				total: 2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{
					"Cluster":  testData[1]["Cluster"],
					"Severity": testData[1]["Severity"]},
				total: 1,
			},
		},
		"test_Test_aggregator_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[0]["Namespace"]},
				total:  2,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": testData[1]["Namespace"]},
				total:  1,
			},
		}}, a.result)

	a.count(testFinding(1))

	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		"test_Test_aggregator_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[0]["Cluster"], "Severity": testData[0]["Severity"]},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": testData[1]["Cluster"], "Severity": testData[1]["Severity"]},
				total:  2,
			},
		},
		"test_Test_aggregator_metric2": {
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

func Test_makeAggregationKey(t *testing.T) {
	testMetric := map[Label]string{
		"Cluster":   "value",
		"IsFixable": "false",
	}
	getter := func(label Label) string {
		return testMetric[label]
	}
	key, labels := makeAggregationKey(
		[]Label{"Cluster", "IsFixable"},
		getter,
		testLabelOrder)

	assert.Equal(t, aggregationKey("value|false"), key)
	assert.Equal(t, prometheus.Labels{
		"Cluster":   "value",
		"IsFixable": "false",
	}, labels)

}

func Test_collectMatchingLabels(t *testing.T) {
	i := 0
	for range collectMatchingLabels([]Label{"label1", "label2", "label3"},
		func(l Label) string {
			i++
			return "value"
		}) {
		break
	}
	assert.Equal(t, 1, i)
}

type withIncrement struct {
	n int
}

func (f *withIncrement) GetIncrement() int { return f.n }

var _ WithIncrement = (*withIncrement)(nil)

func TestFinding_GetIncrement(t *testing.T) {
	var f withIncrement
	f.n = 5

	getters := map[Label]func(*withIncrement) string{
		"l1": func(tf *withIncrement) string { return "v1" },
	}
	a := makeAggregator(
		MetricDescriptors{"m1": []Label{"l1"}},
		map[Label]int{"l1": 0}, getters)
	a.count(&f)
	f.n = 7
	a.count(&f)

	assert.Equal(t, 12, a.result["m1"]["v1"].total)
}
