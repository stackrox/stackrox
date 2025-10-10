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

	if len(from.GetRegistries()) > 0 {
		regs = make([]*v1.DelegatedRegistryConfig_DelegatedRegistry, len(from.GetRegistries()))

		for i, reg := range from.GetRegistries() {
			clusterId := reg.GetClusterId()
			path := reg.GetPath()
			regs[i] = v1.DelegatedRegistryConfig_DelegatedRegistry_builder{
				ClusterId: &clusterId,
				Path:      &path,
			}.Build()
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := v1.DelegatedRegistryConfig_EnabledFor(v1.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()])
	defaultClusterId := from.GetDefaultClusterId()

	return v1.DelegatedRegistryConfig_builder{
		EnabledFor:       &enabledFor,
		DefaultClusterId: &defaultClusterId,
		Registries:       regs,
	}.Build()
}

// PublicAPIToStorage converts a delegated registry config from the type used by the gRPC/REST API
// to the type used by storage (db).
func PublicAPIToStorage(from *v1.DelegatedRegistryConfig) *storage.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*storage.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.GetRegistries()) > 0 {
		regs = make([]*storage.DelegatedRegistryConfig_DelegatedRegistry, len(from.GetRegistries()))

		for i, reg := range from.GetRegistries() {
			clusterId := reg.GetClusterId()
			path := reg.GetPath()
			regs[i] = storage.DelegatedRegistryConfig_DelegatedRegistry_builder{
				ClusterId: &clusterId,
				Path:      &path,
			}.Build()
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := storage.DelegatedRegistryConfig_EnabledFor(storage.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()])
	defaultClusterId := from.GetDefaultClusterId()

	return storage.DelegatedRegistryConfig_builder{
		EnabledFor:       &enabledFor,
		DefaultClusterId: &defaultClusterId,
		Registries:       regs,
	}.Build()
}

// PublicAPIToInternalAPI converts a delegated registry config from the type used by the gRPC/REST API
// to the type used by central/sensor inner apis.
func PublicAPIToInternalAPI(from *v1.DelegatedRegistryConfig) *central.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*central.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.GetRegistries()) > 0 {
		regs = make([]*central.DelegatedRegistryConfig_DelegatedRegistry, len(from.GetRegistries()))

		for i, reg := range from.GetRegistries() {
			path := reg.GetPath()
			regs[i] = central.DelegatedRegistryConfig_DelegatedRegistry_builder{
				Path: &path,
			}.Build()
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := central.DelegatedRegistryConfig_EnabledFor(storage.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()])

	return central.DelegatedRegistryConfig_builder{
		EnabledFor: &enabledFor,
		Registries: regs,
	}.Build()
}

// StorageToInternalAPI converts a delegated registry config from the type used by the storage (db) to
// the type used by central/sensor inner apis.
func StorageToInternalAPI(from *storage.DelegatedRegistryConfig) *central.DelegatedRegistryConfig {
	if from == nil {
		return nil
	}

	var regs []*central.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.GetRegistries()) > 0 {
		regs = make([]*central.DelegatedRegistryConfig_DelegatedRegistry, len(from.GetRegistries()))

		for i, reg := range from.GetRegistries() {
			path := reg.GetPath()
			regs[i] = central.DelegatedRegistryConfig_DelegatedRegistry_builder{
				Path: &path,
			}.Build()
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := central.DelegatedRegistryConfig_EnabledFor(v1.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()])

	return central.DelegatedRegistryConfig_builder{
		EnabledFor: &enabledFor,
		Registries: regs,
	}.Build()
}
