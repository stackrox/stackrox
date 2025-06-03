package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseMetricLabels(t *testing.T) {
	config := makeTestMetricLabels(t)
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricLabelExpression(t), labelExpression)
}

func Test_noLabels(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_Labels{
		"metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{},
		},
		"metric2": {},
		"metric3": nil,
	}
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpression)

	labelExpression, err = parseMetricLabels(nil, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpression)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_Labels{
		"metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"unknown": nil,
			},
		},
	}
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "metric1" is not in the list of known labels: [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, labelExpression)

	delete(config, "metric1")
	config["met rick"] = nil
	labelExpression, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: invalid metric name "met rick": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
	assert.Empty(t, labelExpression)

	delete(config, "met rick")
	config["metric1"] = &storage.PrometheusMetricsConfig_Labels{
		Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
			"test": {
				Expression: []*storage.PrometheusMetricsConfig_Labels_Expression_Condition{
					{
						Operator: "smooth",
						Argument: "y",
					},
				},
			},
		},
	}
	labelExpression, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: failed to parse a condition for metric "metric1" with label "test": operator in "smoothy" is not one of ["=" "!=" ">" ">=" "<" "<=" "OR"]`, err.Error())
	assert.Empty(t, labelExpression)
}
