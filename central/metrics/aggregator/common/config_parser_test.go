package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseMetricLabels(t *testing.T) {
	config := makeTestMetricLabels()
	labelExpressions, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, MetricLabelExpressions{
		"metric1": {
			"Severity": {
				MustMakeExpression("=", "CRITICAL*"),
				MustMakeExpression("=", "HIGH*"),
				MustMakeExpression("OR", ""),
				MustMakeExpression("=", "LOW*"),
			},
			"Cluster": nil,
		},
		"metric2": {
			"Namespace": nil,
		},
	}, labelExpressions)
}

func TestReconfigure(t *testing.T) {
	tracker, err := Reconfigure(nil, "test", 0, nil, testLabelOrder)
	assert.NotNil(t, tracker)
	assert.NoError(t, err)
}

func makeTestMetricLabels() map[string]*storage.PrometheusMetricsConfig_LabelExpressions {
	return map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
		"metric1": {
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_LabelExpressions_Expressions{
				"Severity": {
					Expression: []*storage.PrometheusMetricsConfig_LabelExpressions_Expressions_Expression{
						{
							Operator: "=",
							Argument: "CRITICAL*",
						}, {
							Operator: "=",
							Argument: "HIGH*",
						}, {
							Operator: "OR",
						}, {
							Operator: "=",
							Argument: "LOW*",
						},
					},
				},
				"Cluster": nil,
			},
		},
		"metric2": {
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_LabelExpressions_Expressions{
				"Namespace": {},
			},
		},
	}
}
