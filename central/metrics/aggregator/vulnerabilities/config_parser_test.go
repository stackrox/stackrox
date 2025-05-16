package vulnerabilities

import (
	"testing"
	"time"

	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseVulnerabilitiesConfig(t *testing.T) {
	config := makeTestConfig_Vulnerabilities()
	labelExpressions, period, err := parseVulnerabilitiesConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, 42*time.Hour, period)
	assert.Equal(t, common.MetricsConfig{
		"metric1": {
			"Severity": {
				common.MustMakeExpression("=", "CRITICAL*"),
				common.MustMakeExpression("=", "HIGH*"),
				common.MustMakeExpression("OR", ""),
				common.MustMakeExpression("=", "LOW*"),
			},
			"Cluster": nil,
		},
	}, labelExpressions)
}

func Test_reloadVulnerabilityTrackerConfig(t *testing.T) {
	tracker, err := Reconfigure(nil)
	assert.NotNil(t, tracker)
	assert.NoError(t, err)
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
