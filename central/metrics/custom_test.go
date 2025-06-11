package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKnownMetric(t *testing.T) {
	GetExternalRegistry("metrics_test1")
	assert.True(t, IsKnownRegistry(""))
	assert.True(t, IsKnownRegistry("metrics_test1"))
	assert.False(t, IsKnownRegistry("metrics_test2"))
}
