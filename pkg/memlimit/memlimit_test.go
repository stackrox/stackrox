package memlimit

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/size"
	"github.com/stretchr/testify/assert"
)

func TestSetMemoryLimit_ROX_MEMLIMIT(t *testing.T) {
	// 4Gi
	total := 4 * size.GB
	// ~3.8Gi
	expected := int64(4_080_218_932)

	var limit int64
	setMemoryLimit = func(l int64) int64 {
		limit = l
		return l
	}

	// Valid ROX_MEMLIMIT should set the limit to 95% of the request.
	t.Setenv("ROX_MEMLIMIT", strconv.Itoa(total))
	SetMemoryLimit()
	assert.Equal(t, expected, limit)

	// Invalid ROX_MEMLIMIT should keep current limit.
	t.Setenv("ROX_MEMLIMIT", "5Gi")
	SetMemoryLimit()
	assert.Equal(t, expected, limit)
}
