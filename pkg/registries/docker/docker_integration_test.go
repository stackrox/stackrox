package docker

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/require"
)

var (
	// DefaultRegistry defaults to dockerhub
	defaultRegistry = "https://registry-1.docker.io" // variable so that it could be potentially changed
)

func TestGetMetadataIntegration(t *testing.T) {
	dockerHubClient, err := newRegistry(&v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Docker{
			Docker: &v1.DockerConfig{
				Endpoint: defaultRegistry,
			},
		},
	})
	require.NoError(t, err)

	image := v1.Image{
		Id: "sha256:2fa968a4b4013c2521115f6dde277958cf03229b95f13a0c8df831d3eca1aa61",
		Name: &v1.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.14.0",
		},
	}
	_, err = dockerHubClient.Metadata(&image)
	require.Nil(t, err)
}
