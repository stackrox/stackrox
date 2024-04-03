//go:build integration

package docker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

func TestGetMetadataIntegration(t *testing.T) {
	dockerHubClient, err := NewDockerRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://registry.k8s.io",
			},
		},
	}, false)
	require.NoError(t, err)

	image := storage.Image{
		Id: "sha256:93c827f018cf3322f1ff2aa80324a0306048b0a69bc274e423071fb0d2d29d8b",
		Name: &storage.ImageName{
			Registry: "registry.k8s.io",
			Remote:   "k8s-dns-dnsmasq-nanny-amd64",
			Tag:      "1.14.8",
		},
	}
	_, err = dockerHubClient.Metadata(&image)
	require.Nil(t, err)
}

func TestOCIImageIndexManifest(t *testing.T) {
	gcrClient, err := NewDockerRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://gcr.io",
			},
		},
	}, false)
	require.NoError(t, err)

	image := storage.Image{
		Id: "sha256:a01d47d4036cae5a67a9619e3d06fa14a6811a2247b4da72b4233ece4efebd57",
		Name: &storage.ImageName{
			Registry: "gcr.io",
			Remote:   "distroless/static-debian11",
			Tag:      "latest",
		},
	}

	_, err = gcrClient.Metadata(&image)
	require.NoError(t, err)
}

func TestOCIImageIndexManifestWithoutManifestCall(t *testing.T) {
	gcrClient, err := NewRegistryWithoutManifestCall(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://gcr.io",
			},
		},
	}, false)
	require.NoError(t, err)

	image := storage.Image{
		Id: "sha256:a01d47d4036cae5a67a9619e3d06fa14a6811a2247b4da72b4233ece4efebd57",
		Name: &storage.ImageName{
			Registry: "gcr.io",
			Remote:   "distroless/static-debian11",
			Tag:      "latest",
		},
	}

	_, err = gcrClient.Metadata(&image)
	require.NoError(t, err)
}
