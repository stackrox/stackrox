package crud

import "github.com/stackrox/rox/pkg/features"

// DeleterOption represents an option on a created Deleter.
type DeleterOption func(*deleterImpl)

// RemoveFromIndex removes the key from index after deletion. Happens lazily, so propagation may not be immediate.
func RemoveFromIndex() DeleterOption {
	return func(rc *deleterImpl) {
		rc.removeFromIndex = true
	}
}

// RemoveFromIndexIfFeatureEnabled removes the key from index after deletion if feature is enabled. Happens lazily, so propagation may not be immediate.
func RemoveFromIndexIfFeatureEnabled(featureFlag features.FeatureFlag) DeleterOption {
	if !featureFlag.Enabled() {
		return func(impl *deleterImpl) {}
	}
	return func(rc *deleterImpl) {
		rc.removeFromIndex = true
	}
}

// Shared causes the object to only be removed if all references to it in the graph have been removed.
func Shared() DeleterOption {
	return func(rc *deleterImpl) {
		rc.shared = true
	}
}

// WithDeleterCache adds the passed cache to the deleter
func WithDeleterCache(c *Cache) DeleterOption {
	return func(dc *deleterImpl) {
		dc.cache = c
	}
}
