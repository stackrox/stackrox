// +build integration

package anchore

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stretchr/testify/require"
)

const (
	anchoreURLEnv = "ROX_ANCHORE_URL"
)

func TestAnchore(t *testing.T) {
	if os.Getenv(anchoreURLEnv) == "" {
		t.Skipf("Skipping Anchore integration test")
	}

	registryFactory := registries.NewFactory()
	registrySet := registries.NewSet(registryFactory)

	err := registrySet.UpdateImageIntegration(&storage.ImageIntegration{
		Id:   "id",
		Name: "dockerhub",
		Type: "docker",
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "k8s.gcr.io",
			},
		},
	})
	require.NoError(t, err)

	anchore, err := newScanner(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Anchore{
			Anchore: &storage.AnchoreConfig{
				Endpoint: os.Getenv(anchoreURLEnv),
				Username: "admin",
				Password: "foobar",
			},
		},
	}, registrySet)
	require.NoError(t, err)

	scan, err := anchore.GetLastScan(&storage.Image{
		Id: "sha256:23df717980b4aa08d2da6c4cfa327f1b730d92ec9cf740959d2d5911830d82fb",
		Name: &storage.ImageName{
			Registry: "k8s.gcr.io",
			Remote:   "k8s-dns-sidecar-amd64",
			Tag:      "1.14.8",
			FullName: "k8s.gcr.io/k8s-dns-sidecar-amd64:1.14.8",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, scan)
}
