package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetry_EventualSuccess(t *testing.T) {
	const maxFailCount = 10
	retryCount := 0

	Retry(t, maxFailCount, 0, func(t T) {
		retryCount++
		assert.Equal(t, maxFailCount, retryCount)
	})
}

func TestRetry_PanicPassThrough(t *testing.T) {
	assert.PanicsWithValue(t, "foo", func() {
		retryCount := 0
		Retry(t, 2, 0, func(t T) {
			retryCount++
			if retryCount == 1 {
				panic("foo")
			}
		})
	})
}
