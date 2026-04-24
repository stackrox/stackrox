package clusterlabels

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SetAndGet(t *testing.T) {
	store := NewStore()

	// Initially empty
	labels := store.Get()
	assert.Empty(t, labels)

	// Set labels
	testLabels := map[string]string{
		"environment": "production",
		"region":      "us-east-1",
		"team":        "platform",
	}
	store.Set(testLabels)

	// Get should return the labels
	labels = store.Get()
	assert.Equal(t, testLabels, labels)

	// Update labels
	updatedLabels := map[string]string{
		"environment": "staging",
		"region":      "us-west-2",
	}
	store.Set(updatedLabels)

	labels = store.Get()
	assert.Equal(t, updatedLabels, labels)
}

func TestStore_GetClusterLabels(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	testLabels := map[string]string{
		"cluster-name": "prod-cluster-1",
		"cloud":        "aws",
	}
	store.Set(testLabels)

	// GetClusterLabels should return the same as Get
	labels, err := store.GetClusterLabels(ctx, "cluster-id")
	require.NoError(t, err)
	assert.Equal(t, testLabels, labels)

	// Should work with any cluster ID (parameters are ignored)
	labels, err = store.GetClusterLabels(ctx, "different-cluster-id")
	require.NoError(t, err)
	assert.Equal(t, testLabels, labels)
}

func TestStore_EmptyLabels(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Get on empty store
	labels := store.Get()
	assert.NotNil(t, labels)
	assert.Empty(t, labels)

	// GetClusterLabels on empty store
	labels, err := store.GetClusterLabels(ctx, "cluster-id")
	require.NoError(t, err)
	assert.NotNil(t, labels)
	assert.Empty(t, labels)
}
