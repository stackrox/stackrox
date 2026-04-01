package resources

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceStore_GetNamespaceLabels(t *testing.T) {
	ctx := context.Background()
	depStore := NewDeploymentStore(nil)
	podStore := NewPodStore()
	store := NewNamespaceStore(depStore, podStore)

	ns := &storage.NamespaceMetadata{
		Id:   "ns-123",
		Name: "test-namespace",
		Labels: map[string]string{
			"app":  "myapp",
			"tier": "frontend",
		},
	}

	// Process create event
	store.ProcessEvent(central.ResourceAction_CREATE_RESOURCE, ns)

	// Verify labels can be retrieved by name
	labels, err := store.GetNamespaceLabels(ctx, "cluster-id", "test-namespace")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app": "myapp", "tier": "frontend"}, labels)

	// Verify not found case
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "nonexistent-namespace")
	require.NoError(t, err)
	assert.Nil(t, labels)

	// Process update event with different labels
	updatedNs := &storage.NamespaceMetadata{
		Id:   "ns-123",
		Name: "test-namespace",
		Labels: map[string]string{
			"app":  "myapp",
			"tier": "backend",
		},
	}
	store.ProcessEvent(central.ResourceAction_UPDATE_RESOURCE, updatedNs)

	// Verify updated labels
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "test-namespace")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app": "myapp", "tier": "backend"}, labels)

	// Process remove event
	store.ProcessEvent(central.ResourceAction_REMOVE_RESOURCE, updatedNs)

	// Verify namespace is removed
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "test-namespace")
	require.NoError(t, err)
	assert.Nil(t, labels)
}
