package resources

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceStore_GetNamespaceLabels(t *testing.T) {
	store := newNamespaceStore()
	ctx := context.Background()

	ns1 := &storage.NamespaceMetadata{
		Id:   "ns-123",
		Name: "namespace-1",
		Labels: map[string]string{
			"app": "frontend",
			"env": "prod",
		},
	}

	ns2 := &storage.NamespaceMetadata{
		Id:   "ns-456",
		Name: "namespace-2",
		Labels: map[string]string{
			"app": "backend",
			"env": "staging",
		},
	}

	// Add namespaces
	store.addNamespace(ns1)
	store.addNamespace(ns2)

	// Test lookup by name for ns1
	labels, err := store.GetNamespaceLabels(ctx, "cluster-id", "namespace-1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app": "frontend", "env": "prod"}, labels)

	// Test lookup by name for ns2
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "namespace-2")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app": "backend", "env": "staging"}, labels)

	// Test not found
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "non-existent")
	require.NoError(t, err)
	assert.Nil(t, labels)

	// Remove ns1 and verify
	store.removeNamespace(ns1)
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "namespace-1")
	require.NoError(t, err)
	assert.Nil(t, labels)

	// ns2 should still be there
	labels, err = store.GetNamespaceLabels(ctx, "cluster-id", "namespace-2")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app": "backend", "env": "staging"}, labels)
}
