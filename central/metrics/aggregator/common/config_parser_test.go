package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseMetricLabels(t *testing.T) {
	config := makeTestMetricLabels(t)
	labelExpressions, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricLabelExpressions(t), labelExpressions)
}
