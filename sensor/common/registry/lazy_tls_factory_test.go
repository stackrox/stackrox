package registry

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRegistryErrors(t *testing.T) {
	factory := newLazyFactory(nil)

	tcs := []struct {
		desc    string
		wantErr string
		source  *storage.ImageIntegration
	}{
		{"nil source", "integration is nil", nil},
		{"unknown type", "registry with type", &storage.ImageIntegration{
			Type: "fake",
		}},
		{"nil docker config", "docker config is nil", &storage.ImageIntegration{
			Type:              types.DockerType,
			IntegrationConfig: &storage.ImageIntegration_Docker{},
		}},
		{"empty registry host", "empty registry host", &storage.ImageIntegration{
			Type: types.DockerType,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{},
			},
		}},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			reg, err := factory.CreateRegistry(tc.source)
			require.ErrorContains(t, err, tc.wantErr)
			assert.Nil(t, reg)
		})
	}

}
