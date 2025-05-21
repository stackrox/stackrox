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

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetricsConfig_LabelExpressions {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
		pfx + "_metric1": {
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_LabelExpressions_Expressions{
				"Severity": {
					Expression: []*storage.PrometheusMetricsConfig_LabelExpressions_Expressions_Expression{
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
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_LabelExpressions_Expressions{
				"Namespace": {},
			},
		},
	}
}

func makeTestMetricLabelExpressions(t *testing.T) MetricLabelsExpressions {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricLabelsExpressions{
		pfx + "_metric1": {
			"Severity": {
				MustMakeExpression("=", "CRITICAL*"),
				MustMakeExpression("OR", ""),
				MustMakeExpression("=", "HIGH*"),
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
		"not good":         "bad characters",
		"":                 "empty",
		"abc_defAZ0145609": "",
		"not-good":         "bad characters",
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
