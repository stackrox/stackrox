// +build integration

package tenable

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	tenableRegistry "github.com/stackrox/rox/pkg/registries/tenable"
	"github.com/stretchr/testify/require"
)

func TestTenable(t *testing.T) {
	t.Skip("We only have a trial so don't actually run this during QA tests")
	return
	protoImageIntegration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Tenable{
			Tenable: &storage.TenableConfig{
				AccessKey: os.Getenv("ACCESS_KEY"),
				SecretKey: os.Getenv("SECRET_KEY"),
			},
		},
	}

	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "registry.cloud.tenable.com",
			Remote:   "nginx/nginx",
			Tag:      "1.10",
		},
	}

	_, creator := tenableRegistry.Creator()
	registry, err := creator(protoImageIntegration)
	require.NoError(t, err)

	image.Metadata, err = registry.Metadata(image)
	require.NoError(t, err)
	require.NotNil(t, image.Metadata)

	scanner, err := newScanner(protoImageIntegration)
	require.NoError(t, err)
	require.NoError(t, scanner.Test())

	scan, err := scanner.GetLastScan(image)
	require.NoError(t, err)
	require.NotNil(t, scan)
}
