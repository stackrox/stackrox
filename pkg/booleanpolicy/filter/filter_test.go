package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluationFilter_ZeroValue(t *testing.T) {
	var f EvaluationFilter
	assert.False(t, f.IsNonDefault())
}
