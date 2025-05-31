package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
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

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetricsConfig_Labels {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetricsConfig_Labels{
		pfx + "_metric1": {
			LabelExpression: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"Severity": {
					Expression: []*storage.PrometheusMetricsConfig_Labels_Expression_Condition{
						{
							Operator: "=",
							Argument: "CRITICAL*",
						}, {
							Operator: "OR",
						}, {
							Operator: "=",
							Argument: "HIGH*",
						},
					},
				},
				"Cluster": nil,
			},
		},
		pfx + "_metric2": {
			LabelExpression: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
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
				MustMakeCondition("OR", ""),
				MustMakeCondition("=", "HIGH*"),
			},
			"Cluster": nil,
		},
		pfx + "_metric2": {
			"Namespace": nil,
		},
	}
}

func Test_validateMetricName(t *testing.T) {
	tests := map[string]string{
		"good":             "",
		"not good":         `doesn't match "^[a-zA-Z0-9_]+$"`,
		"":                 "empty",
		"abc_defAZ0145609": "",
		"not-good":         `doesn't match "^[a-zA-Z0-9_]+$"`,
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateMetricName(name); err != nil {
				assert.Equal(t, expected, err.Error())
			} else {
				assert.Empty(t, expected)
			}
		})
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
