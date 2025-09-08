package rendercache

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newMockObject(uid string) ctrlClient.Object {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
			UID:       types.UID(uid),
		},
	}
}

func TestRenderCache_BasicOperations(t *testing.T) {
	cache := NewRenderCache()
	obj := newMockObject("test-uid")
	expectedHash := "test-hash"

	cache.SetCAHash(obj, expectedHash)
	retrievedHash, found := cache.GetCAHash(obj)
	if !found {
		t.Error("Expected to find hash after setting it")
	}
	if retrievedHash != expectedHash {
		t.Errorf("Expected CAHash %s, got %s", expectedHash, retrievedHash)
	}

	_, found = cache.GetCAHash(newMockObject("non-existent"))
	if found {
		t.Error("Expected not to find hash for non-existent key")
	}

	cache.Delete(obj)
	_, found = cache.GetCAHash(obj)
	if found {
		t.Error("Expected not to find hash after deletion")
	}
}

func TestRenderCache_Clear(t *testing.T) {
	cache := NewRenderCache()
	obj1 := newMockObject("test-uid-1")
	obj2 := newMockObject("test-uid-2")

	cache.SetCAHash(obj1, "test-hash-1")
	cache.SetCAHash(obj2, "test-hash-2")

	_, found1 := cache.GetCAHash(obj1)
	_, found2 := cache.GetCAHash(obj2)
	if !found1 || !found2 {
		t.Error("Expected to find both hashes before clear")
	}

	cache.Clear()
	_, found1 = cache.GetCAHash(obj1)
	_, found2 = cache.GetCAHash(obj2)
	if found1 || found2 {
		t.Error("Expected not to find any hashes after clear")
	}
}

func TestRenderCache_Update(t *testing.T) {
	cache := NewRenderCache()
	obj := newMockObject("test-uid")
	originalHash := "original-hash"
	updatedHash := "updated-hash"

	cache.SetCAHash(obj, originalHash)
	retrieved, found := cache.GetCAHash(obj)
	if !found || retrieved != originalHash {
		t.Error("Failed to set original hash")
	}

	cache.SetCAHash(obj, updatedHash)
	retrieved, found = cache.GetCAHash(obj)
	if !found || retrieved != updatedHash {
		t.Error("Failed to update hash")
	}
}
