package nexus

import (
	"os"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

// This requires a Nexus Sonatype registry with a proxy to Dockerhub
func TestNexus(t *testing.T) {
	endpoint := os.Getenv("NEXUS_ENDPOINT")
	if endpoint == "" {
		t.Skipf("ENDPOINT is required for Nexus integration test")
	}
	dockerHubClient, err := newRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: endpoint,
				Username: "admin",
				Password: "admin123",
			},
		},
	})
	require.NoError(t, err)

	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	image := storage.Image{
		Id: "sha256:e2847e35d4e0e2d459a7696538cbfea42ea2d3b8a1ee8329ba7e68694950afd3",
		Name: &storage.ImageName{
			Registry: endpoint,
			Remote:   "nginx",
			Tag:      "latest",
		},
	}
	_, err = dockerHubClient.Metadata(&image)
	require.NoError(t, err)
}
