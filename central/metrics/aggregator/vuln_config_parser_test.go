package aggregator

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseVulnerabilitiesConfig(t *testing.T) {
	config := makeTestConfig_Vulnerabilities()
	labelExpressions, period, err := parseVulnerabilitiesConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, 42*time.Hour, period)
	assert.Equal(t, metricsConfig{
		"metric1": {
			"Severity": {
				{"=", "CRITICAL*"},
				{"=", "HIGH*"},
				{op: "OR"},
				{"=", "LOW*"},
			},
			"Cluster": nil,
		},
	}, labelExpressions)
}

func makeTestConfig_Vulnerabilities() *storage.PrometheusMetricsConfig_Vulnerabilities {
	return &storage.PrometheusMetricsConfig_Vulnerabilities{
		GatheringPeriodHours: 42,
		MetricLabels: map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
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
		},
	}
}
