package cache

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

// Image is a cache for scanned images
type Image expiringcache.Cache[Key, Value]

// Value represents the value stored in image cache
type Value interface {
	// WaitAndGet wait for image scan before returning the image
	WaitAndGet() *storage.Image
	// GetIfDone returns image if scan was done, otherwise it returns nil
	GetIfDone() *storage.Image
}

// Key is the type for keys in image cache to prevent accidental use,
// it should be obtained from image with GetKey
type Key string

// KeyProvider represents an interface from which image cache can be generated.
type KeyProvider interface {
	GetId() string
	GetName() *storage.ImageName
}

// GetKey generates image cache key from a cache key provider.
func GetKey(provider KeyProvider) Key {
	if id := provider.GetId(); id != "" {
		return Key(id)
	}
	return Key(provider.GetName().GetFullName())
}

// CompareKeys given two KeyProvider, compares if they're equal
func CompareKeys(a, b KeyProvider) bool {
	if a.GetId() != "" && b.GetId() != "" {
		return a.GetId() == b.GetId()
	}
	return a.GetName().GetFullName() == b.GetName().GetFullName()
}
