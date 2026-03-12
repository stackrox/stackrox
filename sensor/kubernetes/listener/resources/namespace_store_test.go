package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceStore_LookupNamespaceLabelsByID(t *testing.T) {
	store := newNamespaceStore()

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

	// Test lookup by ID for ns1
	labels, found := store.LookupNamespaceLabelsByID("ns-123")
	assert.True(t, found)
	assert.Equal(t, map[string]string{"app": "frontend", "env": "prod"}, labels)

	// Test lookup by ID for ns2
	labels, found = store.LookupNamespaceLabelsByID("ns-456")
	assert.True(t, found)
	assert.Equal(t, map[string]string{"app": "backend", "env": "staging"}, labels)

	// Test not found
	labels, found = store.LookupNamespaceLabelsByID("ns-999")
	assert.False(t, found)
	assert.Nil(t, labels)

	// Remove ns1 and verify
	store.removeNamespace(ns1)
	labels, found = store.LookupNamespaceLabelsByID("ns-123")
	assert.False(t, found)
	assert.Nil(t, labels)

	// ns2 should still be there
	labels, found = store.LookupNamespaceLabelsByID("ns-456")
	assert.True(t, found)
	assert.Equal(t, map[string]string{"app": "backend", "env": "staging"}, labels)
}
