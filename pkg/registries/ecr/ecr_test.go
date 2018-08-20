// +build integration

package ecr

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECRIntegration(t *testing.T) {
	accessID := os.Getenv("ACCESS_ID")
	accessKey := os.Getenv("ACCESS_KEY")
	registryID := os.Getenv("REGISTRY_ID")
	integration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Ecr{
			Ecr: &v1.ECRConfig{
				Region:          "us-west-2",
				RegistryId:      registryID,
				AccessKeyId:     accessID,
				SecretAccessKey: accessKey,
			},
		},
	}
	ecr, err := newRegistry(integration)
	require.NoError(t, err)

	assert.NoError(t, ecr.Test())

	metadata, err := ecr.Metadata(&v1.Image{
		Name: &v1.ImageName{
			Registry: fmt.Sprintf("%s.dkr.ecr.us-west-2.amazonaws.com", registryID),
			Remote:   "testing",
			Tag:      "latest",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, metadata)
}
