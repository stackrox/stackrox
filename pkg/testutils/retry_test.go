package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_EventualSuccess(t *testing.T) {
	const maxFailCount = 10
	retryCount := 0

	Retry(t, maxFailCount, 0, func(t T) {
		retryCount++
		assert.Equal(t, maxFailCount, retryCount)
	})
}

func TestRetry_StopsWhenNoLongerAsserting(t *testing.T) {
	retryCount := 0

	Retry(t, 10, 0, func(t T) {
		retryCount++
		assert.Equal(t, 5, retryCount)
	})
	assert.Equal(t, retryCount, 5)
}

func TestRetry_StopsWhenNoLongerRequiring(t *testing.T) {
	retryCount := 0

	Retry(t, 10, 0, func(t T) {
		retryCount++
		require.Equal(t, 5, retryCount)
	})
	assert.Equal(t, retryCount, 5)
}

func TestRetry_PanicPassThrough(t *testing.T) {
	retryCount := 0
	assert.PanicsWithValue(t, "foo", func() {
		Retry(t, 10, 0, func(t T) {
			retryCount++
			if retryCount == 1 {
				panic("foo")
			}
		})
	})
	assert.Equal(t, retryCount, 1, "did not retry")
}
