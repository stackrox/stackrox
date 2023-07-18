//go:build integration

package ecr

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECRIntegration(t *testing.T) {
	accessID := os.Getenv("ACCESS_ID")
	if accessID == "" {
		t.Skip("ACCESS_ID required for ECR integration test")
		return
	}
	accessKey := os.Getenv("ACCESS_KEY")
	if accessKey == "" {
		t.Skip("ACCESS_KEY required for ECR integration test")
		return
	}
	registryID := os.Getenv("REGISTRY_ID")
	if registryID == "" {
		t.Skip("REGISTRY_ID required for ECR integration test")
		return
	}
	integration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Ecr{
			Ecr: &storage.ECRConfig{
				Region:          "us-west-2",
				RegistryId:      registryID,
				AccessKeyId:     accessID,
				SecretAccessKey: accessKey,
			},
		},
	}
	ecr, err := newRegistry(integration, false)
	require.NoError(t, err)

	assert.NoError(t, ecr.Test())
	assert.Equal(t, fmt.Sprintf("%s.dkr.ecr.us-west-2.amazonaws.com", registryID), ecr.endpoint)

	metadata, err := ecr.Metadata(&storage.Image{
		Name: &storage.ImageName{
			Registry: fmt.Sprintf("%s.dkr.ecr.us-west-2.amazonaws.com", registryID),
			Remote:   "testing",
			Tag:      "latest",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, metadata)
}
