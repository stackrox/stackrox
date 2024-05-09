//go:build integration

package docker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMetadataIntegration(t *testing.T) {
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	metricsHandler := types.NewMetricsHandler("docker")
	dockerHubClient, err := NewDockerRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://registry.k8s.io",
			},
		},
	}, false, metricsHandler)
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

	// Make sure that request and histogram metrics but no timeouts have been recorded.
	assert.NotEmpty(t, metricsHandler.TestCollectRequestCounter(t))
	assert.Empty(t, metricsHandler.TestCollectTimeoutCounter(t))
	assert.NotEmpty(t, metricsHandler.TestCollectHistogramCounter(t))
}

func TestOCIImageIndexManifest(t *testing.T) {
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	gcrClient, err := NewDockerRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://gcr.io",
			},
		},
	}, false, nil)
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
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	gcrClient, err := NewRegistryWithoutManifestCall(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://gcr.io",
			},
		},
	}, false, nil)
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
