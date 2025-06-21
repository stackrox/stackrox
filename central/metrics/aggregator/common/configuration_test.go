package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var testLabelGetters = []LabelGetter[OneOrMore]{
	testDataGetter("test"),
	testDataGetter("Cluster"),
	testDataGetter("Namespace"),
	testDataGetter("CVE"),
	testDataGetter("Severity"),
	testDataGetter("CVSS"),
	testDataGetter("IsFixable"),
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

func testDataGetter(label Label) LabelGetter[OneOrMore] {
	return LabelGetter[OneOrMore]{
		label,
		func(i OneOrMore) string { return testData[i][label] }}
}

var testLabelOrder = MakeLabelOrderMap(testLabelGetters)

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetricsConfig_Labels {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetricsConfig_Labels{
		pfx + "_metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"Severity": {
					Expression: []*storage.PrometheusMetricsConfig_Labels_Expression_Condition{
						{
							Operator: "=",
							Argument: "CRITICAL*",
						},
					},
				},
				"Cluster": nil,
			},
		},
		pfx + "_metric2": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"Namespace": {},
			},
		},
	}
}

func makeTestMetricLabelExpression(t *testing.T) MetricsConfiguration {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricsConfiguration{
		pfx + "_metric1": {
			"Severity": {
				MustMakeCondition("=", "CRITICAL*"),
			},
			"Cluster": nil,
		},
		pfx + "_metric2": {
			"Namespace": nil,
		},
	}
}

func TestHasAnyLabelOf(t *testing.T) {
	mcfg := MetricsConfiguration{
		"metric1": map[Label]Expression{
			"label1": nil,
			"label2": nil,
		},
		"metric2": map[Label]Expression{
			"label3": nil,
			"label4": nil,
		},
	}
	assert.False(t, mcfg.HasAnyLabelOf([]Label{}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label1"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label3"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label0", "label1"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label0", "label4"}))
	assert.False(t, mcfg.HasAnyLabelOf([]Label{"label0", "label5"}))
}
