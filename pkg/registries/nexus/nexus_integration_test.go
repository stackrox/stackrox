package nexus

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/require"
)

// This requires a Nexus Sonatype registry with a proxy to Dockerhub
func TestNexus(t *testing.T) {
	endpoint := os.Getenv("NEXUS_ENDPOINT")
	if endpoint == "" {
		t.Skipf("ENDPOINT is required for Nexus integration test")
	}

	typ, creator := Creator()
	require.Equal(t, types.NexusType, typ)

	reg, err := creator(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: endpoint,
				Username: "admin",
				Password: "admin123",
			},
		},
	}, nil)
	require.NoError(t, err)

	endpoint = urlfmt.TrimHTTPPrefixes(endpoint)
	image := storage.Image{
		Id: "sha256:e2847e35d4e0e2d459a7696538cbfea42ea2d3b8a1ee8329ba7e68694950afd3",
		Name: &storage.ImageName{
			Registry: endpoint,
			Remote:   "nginx",
			Tag:      "latest",
		},
	}
	_, err = reg.Metadata(&image)
	require.NoError(t, err)
}
