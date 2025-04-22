package telemetry

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

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

func Test_parseConfig(t *testing.T) {
	config := &storage.PrometheusMetricsConfig{
		GatheringPeriodHours: 42,
		MetricLabels: map[string]*storage.PrometheusMetricsConfig_LabelExpressions{
			"metric1": {
				LabelExpressions: map[string]*storage.PrometheusMetricsConfig_Expressions{
					"Severity": {
						Expression: []*storage.PrometheusMetricsConfig_Expression{
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
	labelExpressions, period, err := parseConfig(config)
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
