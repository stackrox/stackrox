package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceStore_LabelLookup(t *testing.T) {
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

	// Verify labels can be retrieved by ID
	labels, found := store.LookupNamespaceLabelsByID("ns-123")
	assert.True(t, found)
	assert.Equal(t, map[string]string{"app": "myapp", "tier": "frontend"}, labels)

	// Verify not found case
	labels, found = store.LookupNamespaceLabelsByID("ns-456")
	assert.False(t, found)
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
	labels, found = store.LookupNamespaceLabelsByID("ns-123")
	assert.True(t, found)
	assert.Equal(t, map[string]string{"app": "myapp", "tier": "backend"}, labels)

	// Process remove event
	store.ProcessEvent(central.ResourceAction_REMOVE_RESOURCE, updatedNs)

	// Verify namespace is removed
	labels, found = store.LookupNamespaceLabelsByID("ns-123")
	assert.False(t, found)
	assert.Nil(t, labels)
}
