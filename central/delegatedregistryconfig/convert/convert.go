// Package convert provides utility functions for converting between the various
// DelegatedRegistryConfig types.
//   - "Storage"     (storage.DelegatedRegistryConfig) - for persistance
//   - "PublicAPI"   (v1.DelegatedRegistryConfig)      - for exposed REST/gRPC API
//   - "InternalAPI" (central.DelegatedRegistryConfig) - for Central/Sensor inner API
package convert

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// StorageToPublicAPI converts a delegated registry config from the type used by storage (db) to
// the type used by the gRPC/REST API.
func StorageToPublicAPI(from *storage.DelegatedRegistryConfig) *v1.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*v1.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*v1.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &v1.DelegatedRegistryConfig_DelegatedRegistry{
				ClusterId: reg.ClusterId,
				Path:      reg.Path,
			}
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := v1.DelegatedRegistryConfig_EnabledFor_value[from.EnabledFor.String()]

	return &v1.DelegatedRegistryConfig{
		EnabledFor:       v1.DelegatedRegistryConfig_EnabledFor(enabledFor),
		DefaultClusterId: from.DefaultClusterId,
		Registries:       regs,
	}
}

// PublicAPIToStorage converts a delegated registry config from the type used by the gRPC/REST API
// to the type used by storage (db).
func PublicAPIToStorage(from *v1.DelegatedRegistryConfig) *storage.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*storage.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*storage.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &storage.DelegatedRegistryConfig_DelegatedRegistry{
				ClusterId: reg.ClusterId,
				Path:      reg.Path,
			}
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := storage.DelegatedRegistryConfig_EnabledFor_value[from.EnabledFor.String()]

	return &storage.DelegatedRegistryConfig{
		EnabledFor:       storage.DelegatedRegistryConfig_EnabledFor(enabledFor),
		DefaultClusterId: from.DefaultClusterId,
		Registries:       regs,
	}
}

// PublicAPIToInternalAPI converts a delegated registry config from the type used by the gRPC/REST API
// to the type used by central/sensor inner apis.
func PublicAPIToInternalAPI(from *v1.DelegatedRegistryConfig) *central.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*central.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*central.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &central.DelegatedRegistryConfig_DelegatedRegistry{
				Path: reg.Path,
			}
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := storage.DelegatedRegistryConfig_EnabledFor_value[from.EnabledFor.String()]

	return &central.DelegatedRegistryConfig{
		EnabledFor: central.DelegatedRegistryConfig_EnabledFor(enabledFor),
		Registries: regs,
	}
}

// StorageToInternalAPI converts a delegated registry config from the type used by the storage (db) to
// the type used by central/sensor inner apis.
func StorageToInternalAPI(from *storage.DelegatedRegistryConfig) *central.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*central.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*central.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &central.DelegatedRegistryConfig_DelegatedRegistry{
				Path: reg.Path,
			}
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := v1.DelegatedRegistryConfig_EnabledFor_value[from.EnabledFor.String()]

	return &central.DelegatedRegistryConfig{
		EnabledFor: central.DelegatedRegistryConfig_EnabledFor(enabledFor),
		Registries: regs,
	}
}
