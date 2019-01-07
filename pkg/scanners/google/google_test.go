// +build integration

package google

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

const project = "ultra-current-825"

func TestGoogle(t *testing.T) {
	serviceAccount := os.Getenv("SERVICE_ACCOUNT")
	if serviceAccount == "" {
		t.Skip("SERVICE_ACCOUNT is required for Google integration test")
		return
	}

	integration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Google{
			Google: &storage.GoogleConfig{
				Endpoint:       "us.gcr.io",
				ServiceAccount: os.Getenv("SERVICE_ACCOUNT"),
				Project:        project,
			},
		},
	}

	scanner, err := newScanner(integration)
	if err != nil {
		require.NoError(t, err)
	}
	image := &storage.Image{
		Id: "158d3d219e6efd9c6e25e8b25b5ad04b726880bff6c102973c07bbf5156c7181",
		Name: &storage.ImageName{
			Registry: "us.gcr.io",
			Remote:   project + "/music-nginx",
		},
	}
	scan, err := scanner.GetLastScan(image)
	if err != nil {
		require.NoError(t, err)
	}
	require.NotEmpty(t, scan.GetComponents())
}
