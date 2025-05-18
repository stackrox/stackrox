package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseMetricLabels(t *testing.T) {
	config := makeTestMetricLabels(t)
	labelExpressions, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricLabelExpressions(t), labelExpressions)
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

func makeTestMetricLabelExpressions(t *testing.T) MetricLabelExpressions {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricLabelExpressions{
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
