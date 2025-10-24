//go:build integration

package ibm

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/require"
)

const (
	testImage = "us.icr.io/sr-testing/nginx:1.10"
)

func TestIBM(t *testing.T) {
	if os.Getenv("IBM_CR_READONLY") == "" {
		t.Skip("IBM_CR_READONLY env variable required")
		return
	}
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	reg, err := newRegistry(&storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Ibm{
			Ibm: &storage.IBMRegistryConfig{
				Endpoint: "us.icr.io",
				ApiKey:   os.Getenv("IBM_CR_READONLY"),
			},
		},
	}, false, nil)
	require.NoError(t, err)

	image, err := utils.GenerateImageFromString(testImage)
	require.NoError(t, err)

	metadata, err := reg.Metadata(types.ToImage(image))
	require.NoError(t, err)
	require.NotNil(t, metadata)
}
