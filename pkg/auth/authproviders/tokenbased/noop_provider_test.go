package tokenbased

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stretchr/testify/assert"
)

func TestNoopProvider(t *testing.T) {
	provider := &noopProvider{}
	assert.Empty(t, provider.ID())
	assert.Empty(t, provider.Name())
	assert.Empty(t, provider.Type())
	assert.False(t, provider.Enabled())
	assert.False(t, provider.Active())
	assert.Nil(t, provider.RoleMapper())
	assert.NoError(t, provider.Validate(t.Context(), nil))
	smokeTestUnimplementedFunctions(t, provider, false)
}

func smokeTestUnimplementedFunctions(t *testing.T, source authproviders.Provider, expectedActive bool) {
	sourceInitialID := source.ID()
	assert.Nil(t, source.MergeConfigInto(nil))
	basicConfig := map[string]string{"key": "value"}
	assert.Equal(t, basicConfig, source.MergeConfigInto(basicConfig))
	assert.Nil(t, source.StorageView())
	assert.Nil(t, source.BackendFactory())
	assert.Nil(t, source.Backend())
	backend, err := source.GetOrCreateBackend(t.Context())
	assert.Nil(t, backend)
	assert.NoError(t, err)
	assert.Nil(t, source.Issuer())
	assert.Nil(t, source.AttributeVerifier())
	assert.Nil(t, source.ApplyOptions())
	// Ensure ApplyOption is a noop
	assert.Nil(t, source.ApplyOptions(authproviders.WithID("fakeID")))
	assert.Equal(t, sourceInitialID, source.ID())
	// Ensure MarkAsActive is a noop
	assert.Equal(t, expectedActive, source.Active())
	assert.NoError(t, source.MarkAsActive())
	assert.Equal(t, expectedActive, source.Active())
}
