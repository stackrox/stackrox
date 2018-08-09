// +build integration

package google

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/require"
)

func TestGoogle(t *testing.T) {
	integration := &v1.ImageIntegration{
		Config: map[string]string{
			"serviceAccount": os.Getenv("SERVICE_ACCOUNT"),
			"project":        os.Getenv("PROJECT"),
		},
	}

	scanner, err := newScanner(integration)
	if err != nil {
		require.NoError(t, err)
	}
	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "us.gcr.io",
			Remote:   os.Getenv("PROJECT") + "/music-nginx",
			Sha:      "158d3d219e6efd9c6e25e8b25b5ad04b726880bff6c102973c07bbf5156c7181",
		},
	}
	scan, err := scanner.GetLastScan(image)
	if err != nil {
		require.NoError(t, err)
	}
	require.NotEmpty(t, scan.GetComponents())
}
