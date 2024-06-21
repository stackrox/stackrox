package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTLS(t *testing.T) {
	ctx := context.Background()

	t.Run("secure", func(t *testing.T) {
		c := NewTLSCheckCache(alwaysSecureCheckTLS)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})

	t.Run("insecure", func(t *testing.T) {
		c := NewTLSCheckCache(alwaysInsecureCheckTLS)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		c = NewTLSCheckCache(alwaysInsecureCheckTLS)
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})

	t.Run("error", func(t *testing.T) {
		c := NewTLSCheckCache(alwaysFailCheckTLS)
		secure, skip, err := c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.Error(t, err)
		assert.Len(t, c.results.GetAll(), 1)

		// Results expected to change, skip should be true due to previous error.
		secure, skip, err = c.CheckTLS(ctx, "fake")
		assert.False(t, secure)
		assert.True(t, skip)
		assert.NoError(t, err)
		assert.Len(t, c.results.GetAll(), 1)
	})
}
