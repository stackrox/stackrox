package reposcan

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/registries/types"
)

// RegistryMatcher finds a matching registry for an image name.
// Returns nil if no matching registry is found.
type RegistryMatcher func(imgName *storage.ImageName) types.ImageRegistry

// RegistrySet defines the minimal interface for accessing a collection of registries.
type RegistrySet interface {
	GetAll() []types.ImageRegistry
	GetAllUnique() []types.ImageRegistry
}

// NewRegistryMatcher creates a RegistryMatcher from a RegistrySet.
// The matcher iterates through registries and returns the first match.
func NewRegistryMatcher(set RegistrySet) RegistryMatcher {
	return func(imgName *storage.ImageName) types.ImageRegistry {
		var regs []types.ImageRegistry
		if env.DedupeImageIntegrations.BooleanSetting() {
			regs = set.GetAllUnique()
		} else {
			regs = set.GetAll()
		}
		for _, r := range regs {
			if r.Match(imgName) {
				return r
			}
		}
		return nil
	}
}

// RegistryStore defines the interface for accessing multiple registry sources.
// This is typically implemented by Sensor's registry.Store which manages
// Central-synced integrations, global registries, and pull secrets.
type RegistryStore interface {
	GetCentralRegistries(*storage.ImageName) []types.ImageRegistry
	GetGlobalRegistries(*storage.ImageName) ([]types.ImageRegistry, error)
}

// NewRegistryMatcherFromStore creates a RegistryMatcher that searches multiple sources.
// It tries Central-synced integrations first, then falls back to global registries.
// This is the recommended matcher for Sensor-side scanning.
func NewRegistryMatcherFromStore(store RegistryStore) RegistryMatcher {
	return func(imgName *storage.ImageName) types.ImageRegistry {
		// Try Central-synced integrations first.
		if regs := store.GetCentralRegistries(imgName); len(regs) > 0 {
			for _, r := range regs {
				if r.Match(imgName) {
					return r
				}
			}
		}

		// Fall back to global registries (e.g., OCP global pull secret).
		if regs, err := store.GetGlobalRegistries(imgName); err == nil && len(regs) > 0 {
			for _, r := range regs {
				if r.Match(imgName) {
					return r
				}
			}
		}

		return nil
	}
}
