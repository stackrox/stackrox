//go:build integration
// +build integration

package docker

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/require"
)

func TestGetMetadataIntegration(t *testing.T) {
	dockerHubClient, err := NewDockerRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://k8s.gcr.io",
			},
		},
	})
	require.NoError(t, err)

	image := storage.Image{
		Id: "sha256:93c827f018cf3322f1ff2aa80324a0306048b0a69bc274e423071fb0d2d29d8b",
		Name: &storage.ImageName{
			Registry: "k8s.gcr.io",
			Remote:   "k8s-dns-dnsmasq-nanny-amd64",
			Tag:      "1.14.8",
		},
	}
	_, err = dockerHubClient.Metadata(&image)
	require.Nil(t, err)
}
