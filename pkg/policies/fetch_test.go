package policies

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	t.Skip("Skipping in CI since this is only for local testing / verification")

	f := NewFetcher()

	registryConfig := &types.Config{
		RegistryHostname: "registry-1.docker.io",
	}

	policy, err := f.Fetch(context.Background(), registryConfig, "daha97/policies")
	assert.NoError(t, err)
	assert.NotEmpty(t, policy)
}
