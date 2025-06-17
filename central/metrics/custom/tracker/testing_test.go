package tracker

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

// The test tracker finds some integers to track.
type testFinding int

var testLabelGetters = []LazyLabel[testFinding]{
	testLabel("test"),
	testLabel("Cluster"),
	testLabel("Namespace"),
	testLabel("CVE"),
	testLabel("Severity"),
	testLabel("CVSS"),
	testLabel("IsFixable"),
}

var testLabelOrder = MakeLabelOrderMap(testLabelGetters)

func testLabel(label Label) LazyLabel[testFinding] {
	return LazyLabel[testFinding]{
		label,
		func(i *testFinding) string { return testData[*i][label] }}
}

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

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetrics_Group_Labels {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetrics_Group_Labels{
		pfx + "_metric1": {Labels: []string{"Cluster", "Severity"}},
		pfx + "_metric2": {Labels: []string{"Namespace"}},
	}
}

func makeTestMetricConfiguration(t *testing.T) MetricsConfiguration {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricsConfiguration{
		pfx + "_metric1": {"Cluster", "Severity"},
		pfx + "_metric2": {"Namespace"},
	}
}

func TestMetricsConfiguration_diff(t *testing.T) {
	tests := []struct {
		name         string
		a, b         MetricsConfiguration
		wantToAdd    []MetricName
		wantToDelete []MetricName
		wantChanged  []MetricName
	}{
		{
			name:         "both nil",
			a:            nil,
			b:            nil,
			wantToAdd:    nil,
			wantToDelete: nil,
		},
		{
			name:         "a empty, b has one",
			a:            MetricsConfiguration{},
			b:            MetricsConfiguration{"metric1": {"label1"}},
			wantToAdd:    []MetricName{"metric1"},
			wantToDelete: nil,
		},
		{
			name:         "a has one, b empty",
			a:            MetricsConfiguration{"metric1": {"label1"}},
			b:            MetricsConfiguration{},
			wantToAdd:    nil,
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name:         "a has one, b has another",
			a:            MetricsConfiguration{"metric1": {"label1"}},
			b:            MetricsConfiguration{"metric2": {"label2"}},
			wantToAdd:    []MetricName{"metric2"},
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name: "a and b have overlap",
			a: MetricsConfiguration{
				"metric1": {"label1"},
				"metric2": {"label2"},
			},
			b: MetricsConfiguration{
				"metric2": {"label2"},
				"metric3": {"label3"},
			},
			wantToAdd:    []MetricName{"metric3"},
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name: "identical",
			a: MetricsConfiguration{
				"metric1": {"label1"},
				"metric2": {"label2"},
			},
			b: MetricsConfiguration{
				"metric1": {"label1"},
				"metric2": {"label2"},
			},
			wantToAdd:    nil,
			wantToDelete: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToAdd, gotToDelete, gotChanged := tt.a.diff(tt.b)
			assert.ElementsMatch(t, tt.wantToAdd, gotToAdd)
			assert.ElementsMatch(t, tt.wantToDelete, gotToDelete)
			assert.ElementsMatch(t, tt.wantChanged, gotChanged)
		})
	}
}
