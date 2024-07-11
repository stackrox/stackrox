package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTLS(t *testing.T) {
	ctx := context.Background()

	t.Run("secure", func(t *testing.T) {
		regStore := NewRegistryStore(alwaysSecureCheckTLS)
		secure, skip, err := regStore.checkTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		secure, skip, err = regStore.checkTLS(ctx, "fake")
		assert.True(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)
	})

	t.Run("insecure", func(t *testing.T) {
		regStore := NewRegistryStore(alwaysInsecureCheckTLS)
		secure, skip, err := regStore.checkTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)

		// Ensure the results do not change when attempted again / using cache
		regStore = NewRegistryStore(alwaysInsecureCheckTLS)
		secure, skip, err = regStore.checkTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.NoError(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)
	})

	t.Run("error", func(t *testing.T) {
		regStore := NewRegistryStore(alwaysFailCheckTLS)
		secure, skip, err := regStore.checkTLS(ctx, "fake")
		assert.False(t, secure)
		assert.False(t, skip)
		assert.Error(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)

		// Results expected to change, skip should be true due to previous error.
		secure, skip, err = regStore.checkTLS(ctx, "fake")
		assert.False(t, secure)
		assert.True(t, skip)
		assert.NoError(t, err)
		assert.Len(t, regStore.tlsCheckResults.GetAll(), 1)
	})
}
