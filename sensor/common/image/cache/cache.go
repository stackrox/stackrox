package cache

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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
		if centralcaps.Has(centralsensor.FlattenImageData) {
			return Key(utils.NewImageV2ID(provider.GetName(), id))
		}
		return Key(id)
	}
	return Key(provider.GetName().GetFullName())
}

// KeyFromImageKey derives the canonical cache key from an ImageKey proto,
// applying V2/V1/fullName precedence based on the FlattenImageData capability.
func KeyFromImageKey(k *central.ImageKey) Key {
	if centralcaps.Has(centralsensor.FlattenImageData) {
		if id := k.GetImageIdV2(); id != "" {
			return Key(id)
		}
	} else {
		if id := k.GetImageId(); id != "" {
			return Key(id)
		}
	}
	return Key(k.GetImageFullName())
}

// CompareKeys given two KeyProvider, compares if they're equal
func CompareKeys(a, b KeyProvider) bool {
	if a.GetId() != "" && b.GetId() != "" {
		if centralcaps.Has(centralsensor.FlattenImageData) {
			return utils.NewImageV2ID(a.GetName(), a.GetId()) == utils.NewImageV2ID(b.GetName(), b.GetId())
		}
		return a.GetId() == b.GetId()
	}
	return a.GetName().GetFullName() == b.GetName().GetFullName()
}
