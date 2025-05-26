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
