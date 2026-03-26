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

func TestNamespaceStore_GetAll(t *testing.T) {
	store := newNamespaceStore()

	// Initially empty
	namespaces := store.GetAll()
	assert.Empty(t, namespaces)

	// Add some namespaces
	ns1 := &storage.NamespaceMetadata{
		Id:   "ns-123",
		Name: "namespace-1",
		Labels: map[string]string{
			"app": "frontend",
		},
	}

	ns2 := &storage.NamespaceMetadata{
		Id:   "ns-456",
		Name: "namespace-2",
		Labels: map[string]string{
			"app": "backend",
		},
	}

	ns3 := &storage.NamespaceMetadata{
		Id:   "ns-789",
		Name: "namespace-3",
		Labels: map[string]string{
			"app": "database",
		},
	}

	store.addNamespace(ns1)
	store.addNamespace(ns2)
	store.addNamespace(ns3)

	// GetAll should return all three
	namespaces = store.GetAll()
	assert.Len(t, namespaces, 3)

	// Verify all namespaces are present (order not guaranteed)
	namespaceMap := make(map[string]*storage.NamespaceMetadata)
	for _, ns := range namespaces {
		namespaceMap[ns.Name] = ns
	}

	assert.Equal(t, ns1, namespaceMap["namespace-1"])
	assert.Equal(t, ns2, namespaceMap["namespace-2"])
	assert.Equal(t, ns3, namespaceMap["namespace-3"])

	// Remove one and verify GetAll reflects the change
	store.removeNamespace(ns2)
	namespaces = store.GetAll()
	assert.Len(t, namespaces, 2)

	namespaceMap = make(map[string]*storage.NamespaceMetadata)
	for _, ns := range namespaces {
		namespaceMap[ns.Name] = ns
	}

	assert.Equal(t, ns1, namespaceMap["namespace-1"])
	assert.Equal(t, ns3, namespaceMap["namespace-3"])
	assert.Nil(t, namespaceMap["namespace-2"])
}
