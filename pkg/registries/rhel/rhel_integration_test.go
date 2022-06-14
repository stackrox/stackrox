//go:build integration
// +build integration

package rhel

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRHEL(t *testing.T) {
	_, creator := Creator()
	reg, err := creator(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "registry.access.redhat.com",
			},
		},
	})
	require.NoError(t, err)

	m, err := reg.Metadata(&storage.Image{
		Name: &storage.ImageName{
			Remote: "ubi8/ubi",
			Tag:    "8.3",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, m)
}
