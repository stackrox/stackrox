package service

import (
	"context"
	"slices"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResourcesIncludesGlobalResources(t *testing.T) {
	response, err := (&serviceImpl{}).GetResources(context.Background(), &v1.Empty{})
	require.NoError(t, err)
	assert.Contains(t, response.GetGlobalResources(), "Administration")
	assert.NotContains(t, response.GetGlobalResources(), "Deployment")
	for _, metadata := range resources.ListAllMetadata() {
		assert.Contains(t, response.GetResources(), string(metadata.Resource))
		assert.Equal(t, metadata.GetScope() == permissions.GlobalScope,
			slices.Contains(response.GetGlobalResources(), string(metadata.Resource)))
	}
}
