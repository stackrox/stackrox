// +build integration

package google

import (
	"os"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogle(t *testing.T) {
	if os.Getenv("SERVICE_ACCOUNT") == "" {
		t.Fatal("SERVICE_ACCOUNT env variable required")
	}
	if os.Getenv("PROJECT") == "" {
		t.Fatal("PROJECT env variable required")
	}
	integration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Google{
			Google: &v1.GoogleConfig{
				Endpoint:       "us.gcr.io",
				ServiceAccount: os.Getenv("SERVICE_ACCOUNT"),
			},
		},
	}

	registry, err := newRegistry(integration)
	require.NoError(t, err)

	metadata, err := registry.Metadata(&v1.Image{
		Name: &v1.ImageName{
			Registry: "us.gcr.io",
			Remote:   os.Getenv("PROJECT") + "/music-nginx",
			Tag:      "latest",
		},
	})
	require.NoError(t, err)
	assert.Len(t, metadata.Layers, 14)
}
