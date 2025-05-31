package common

import (
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

func Test_noLabels(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_MetricLabels{
		"metric1": {
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_MetricLabels_Expressions{},
		},
		"metric2": {},
		"metric3": nil,
	}
	labelExpressions, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpressions)

	labelExpressions, err = parseMetricLabels(nil, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpressions)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_MetricLabels{
		"metric1": {
			LabelExpressions: map[string]*storage.PrometheusMetricsConfig_MetricLabels_Expressions{
				"unknown": nil,
			},
		},
	}
	labelExpressions, err := parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "metric1" is not in the list of known labels: [test Cluster Namespace CVE Severity CVSS IsFixable]`, err.Error())
	assert.Empty(t, labelExpressions)

	delete(config, "metric1")
	config["met rick"] = nil
	labelExpressions, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: invalid metric name "met rick": doesn't match "^[a-zA-Z0-9_]+$"`, err.Error())
	assert.Empty(t, labelExpressions)

	delete(config, "met rick")
	config["metric1"] = &storage.PrometheusMetricsConfig_MetricLabels{
		LabelExpressions: map[string]*storage.PrometheusMetricsConfig_MetricLabels_Expressions{
			"test": {
				Expression: []*storage.PrometheusMetricsConfig_MetricLabels_Expressions_Expression{
					{
						Operator: "smooth",
						Argument: "y",
					},
				},
			},
		},
	}
	labelExpressions, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: failed to parse expression for metric "metric1" with label "test": operator in "smoothy" is not one of ["=" "!=" ">" ">=" "<" "<=" "OR"]`, err.Error())
	assert.Empty(t, labelExpressions)
}
