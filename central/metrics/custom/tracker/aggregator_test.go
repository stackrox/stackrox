package tracker

import (
	"regexp"
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
	a := makeAggregator(makeTestMetricDescriptors(t), nil, nil, getters)
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

func Test_filter(t *testing.T) {
	clusterFilter := make(map[Label]*regexp.Regexp)
	clusterFilter[Label("Cluster")] = regexp.MustCompile("^cluster [^5]$")

	severityFilter := make(map[Label]*regexp.Regexp)
	severityFilter[Label("Severity")] = regexp.MustCompile("^CRITICAL|HIGH$")

	incFilters := make(LabelFilters)
	incFilters[MetricName("test_Test_filter_metric1")] = severityFilter
	incFilters[MetricName("test_Test_filter_metric2")] = clusterFilter

	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, incFilters, nil, testLabelGetters)

	// Count all test data:
	for i := range testData {
		a.count(testFinding(i))
	}
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		// "LOW" severity findings are filtered out:
		"test_Test_filter_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": "cluster 1", "Severity": "CRITICAL"},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": "cluster 2", "Severity": "HIGH"},
				total:  1,
			},
		},
		// "cluster 5" findings are filtered out:
		"test_Test_filter_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 1"},
				total:  1,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 2"},
				total:  1,
			},
			"ns 3": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 3"},
				total:  2,
			},
		},
	}, a.result)
}

func Test_excludeFilter(t *testing.T) {
	// Exclude filter to drop LOW severity findings.
	severityExclude := make(map[Label]*regexp.Regexp)
	severityExclude[Label("Severity")] = regexp.MustCompile("^LOW$")

	// Exclude filter to drop cluster 1 findings.
	clusterExclude := make(map[Label]*regexp.Regexp)
	clusterExclude[Label("Cluster")] = regexp.MustCompile("^cluster 1$")

	excFilters := make(LabelFilters)
	excFilters[MetricName("test_Test_excludeFilter_metric1")] = severityExclude
	excFilters[MetricName("test_Test_excludeFilter_metric2")] = clusterExclude

	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, nil, excFilters, testLabelGetters)

	// Count all test data.
	for i := range testData {
		a.count(testFinding(i))
	}
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		// LOW severity findings (indices 2, 4) are excluded.
		"test_Test_excludeFilter_metric1": {
			"cluster 1|CRITICAL": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": "cluster 1", "Severity": "CRITICAL"},
				total:  2,
			},
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": "cluster 2", "Severity": "HIGH"},
				total:  1,
			},
		},
		// cluster 1 findings (indices 0, 3) are excluded.
		"test_Test_excludeFilter_metric2": {
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 2"},
				total:  1,
			},
			"ns 3": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 3"},
				total:  2,
			},
		},
	}, a.result)
}

func Test_includeAndExcludeFilter(t *testing.T) {
	// Include filter to keep only CRITICAL and HIGH severity.
	severityInclude := make(map[Label]*regexp.Regexp)
	severityInclude[Label("Severity")] = regexp.MustCompile("^CRITICAL|HIGH$")

	// Exclude filter to drop cluster 1 findings.
	clusterExclude := make(map[Label]*regexp.Regexp)
	clusterExclude[Label("Cluster")] = regexp.MustCompile("^cluster 1$")

	incFilters := make(LabelFilters)
	incFilters[MetricName("test_Test_includeAndExcludeFilter_metric1")] = severityInclude

	excFilters := make(LabelFilters)
	excFilters[MetricName("test_Test_includeAndExcludeFilter_metric1")] = clusterExclude

	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, incFilters, excFilters, testLabelGetters)

	// Count all test data.
	for i := range testData {
		a.count(testFinding(i))
	}
	assert.Equal(t, map[MetricName]map[aggregationKey]*aggregatedRecord{
		// Only CRITICAL/HIGH kept (include), then cluster 1 dropped (exclude).
		// This leaves only index 1 (cluster 2, HIGH).
		"test_Test_includeAndExcludeFilter_metric1": {
			"cluster 2|HIGH": &aggregatedRecord{
				labels: prometheus.Labels{"Cluster": "cluster 2", "Severity": "HIGH"},
				total:  1,
			},
		},
		// No filters on metric2.
		"test_Test_includeAndExcludeFilter_metric2": {
			"ns 1": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 1"},
				total:  1,
			},
			"ns 2": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 2"},
				total:  1,
			},
			"ns 3": &aggregatedRecord{
				labels: prometheus.Labels{"Namespace": "ns 3"},
				total:  3,
			},
		},
	}, a.result)
}

func Test_makeAggregationKey(t *testing.T) {
	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, nil, nil, testLabelGetters)

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
		MetricDescriptors{"m1": []Label{"l1"}}, nil, nil,
		getters)
	a.count(&f)
	f.n = 7
	a.count(&f)

	assert.Equal(t, 12, a.result["m1"]["v1"].total)
}

func Test_aggregator_reset(t *testing.T) {
	md := makeTestMetricDescriptors(t)
	a := makeAggregator(md, nil, nil, testLabelGetters)

	for i := range testData {
		a.count(testFinding(i))
	}

	assert.NotEmpty(t, a.result["test_Test_aggregator_reset_metric1"])
	assert.NotEmpty(t, a.result["test_Test_aggregator_reset_metric2"])

	a.reset()

	assert.Empty(t, a.result["test_Test_aggregator_reset_metric1"])
	assert.Empty(t, a.result["test_Test_aggregator_reset_metric2"])

	assert.Len(t, a.result, 2)
	assert.Contains(t, a.result, MetricName("test_Test_aggregator_reset_metric1"))
	assert.Contains(t, a.result, MetricName("test_Test_aggregator_reset_metric2"))

	// Verify aggregator works correctly after reset.
	a.count(testFinding(0))
	assert.Len(t, a.result["test_Test_aggregator_reset_metric1"], 1)
	assert.Equal(t, 1, a.result["test_Test_aggregator_reset_metric1"]["cluster 1|CRITICAL"].total)
}
