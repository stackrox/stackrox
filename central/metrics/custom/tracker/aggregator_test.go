package tracker

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func Test_aggregator(t *testing.T) {
	getters := LazyLabelGetters[testFinding]{
		"Severity":  func(tf testFinding) string { return testData[tf]["Severity"] },
		"Cluster":   func(tf testFinding) string { return testData[tf]["Cluster"] },
		"Namespace": func(tf testFinding) string { return testData[tf]["Namespace"] },
	}
	a := makeAggregator(makeTestMetricDescriptors(t), getters)
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
	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, testLabelGetters)

	var metric = MetricName("test_" + t.Name() + "_metric1")
	key, labels := a.makeAggregationKey(
		md[metric],
		testFinding(0),
	)

	assert.Equal(t, aggregationKey("cluster 1|CRITICAL"), key)
	assert.Equal(t, prometheus.Labels{
		"Cluster":  "cluster 1",
		"Severity": "CRITICAL",
	}, labels)

	metric = MetricName("test_" + t.Name() + "_metric2")
	key, labels = a.makeAggregationKey(
		md[metric],
		testFinding(0),
	)

	assert.Equal(t, aggregationKey("ns 1"), key)
	assert.Equal(t, prometheus.Labels{
		"Namespace": "ns 1",
	}, labels)

	metric = MetricName("test_" + t.Name() + "_metric2")
	key, labels = a.makeAggregationKey(
		md[metric],
		testFinding(1),
	)

	assert.Equal(t, aggregationKey("ns 2"), key)
	assert.Equal(t, prometheus.Labels{
		"Namespace": "ns 2",
	}, labels)
}

type withIncrement struct {
	n int
}

func (f *withIncrement) GetIncrement() int { return f.n }

var _ WithIncrement = (*withIncrement)(nil)

func TestFinding_GetIncrement(t *testing.T) {
	var f withIncrement
	f.n = 5

	getters := LazyLabelGetters[*withIncrement]{
		"l1": func(tf *withIncrement) string { return "v1" },
	}
	a := makeAggregator(
		MetricDescriptors{"m1": []Label{"l1"}},
		getters)
	a.count(&f)
	f.n = 7
	a.count(&f)

	assert.Equal(t, 12, a.result["m1"]["v1"].total)
}
