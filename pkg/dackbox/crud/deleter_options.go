package crud

import "github.com/stackrox/stackrox/pkg/features"

// DeleterOption represents an option on a created Deleter.
type DeleterOption func(*deleterImpl)

// RemoveFromIndex removes the key from index after deletion. Happens lazily, so propagation may not be immediate.
func RemoveFromIndex() DeleterOption {
	return func(rc *deleterImpl) {
		rc.removeFromIndex = true
	}
}

// RemoveFromIndexIfAnyFeatureEnabled removes the key from index after deletion if feature is enabled. Happens lazily, so propagation may not be immediate.
func RemoveFromIndexIfAnyFeatureEnabled(featureFlags []features.FeatureFlag) DeleterOption {
	for _, f := range featureFlags {
		if f.Enabled() {
			return func(rc *deleterImpl) {
				rc.removeFromIndex = true
			}
		}
	}
	return func(rc *deleterImpl) {}
}

// Shared causes the object to only be removed if all references to it in the graph have been removed.
func Shared() DeleterOption {
	return func(rc *deleterImpl) {
		rc.shared = true
	}
}
