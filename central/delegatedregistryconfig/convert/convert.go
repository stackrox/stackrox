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
			dd := &v1.DelegatedRegistryConfig_DelegatedRegistry{}
			dd.SetClusterId(reg.GetClusterId())
			dd.SetPath(reg.GetPath())
			regs[i] = dd
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := v1.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()]

	drc := &v1.DelegatedRegistryConfig{}
	drc.SetEnabledFor(v1.DelegatedRegistryConfig_EnabledFor(enabledFor))
	drc.SetDefaultClusterId(from.GetDefaultClusterId())
	drc.SetRegistries(regs)
	return drc
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
			dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
			dd.SetClusterId(reg.GetClusterId())
			dd.SetPath(reg.GetPath())
			regs[i] = dd
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := storage.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()]

	drc := &storage.DelegatedRegistryConfig{}
	drc.SetEnabledFor(storage.DelegatedRegistryConfig_EnabledFor(enabledFor))
	drc.SetDefaultClusterId(from.GetDefaultClusterId())
	drc.SetRegistries(regs)
	return drc
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
			dd := &central.DelegatedRegistryConfig_DelegatedRegistry{}
			dd.SetPath(reg.GetPath())
			regs[i] = dd
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := storage.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()]

	drc := &central.DelegatedRegistryConfig{}
	drc.SetEnabledFor(central.DelegatedRegistryConfig_EnabledFor(enabledFor))
	drc.SetRegistries(regs)
	return drc
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
			dd := &central.DelegatedRegistryConfig_DelegatedRegistry{}
			dd.SetPath(reg.GetPath())
			regs[i] = dd
		}
	}

	// defaults to 0 (NONE) if not found in map
	enabledFor := v1.DelegatedRegistryConfig_EnabledFor_value[from.GetEnabledFor().String()]

	drc := &central.DelegatedRegistryConfig{}
	drc.SetEnabledFor(central.DelegatedRegistryConfig_EnabledFor(enabledFor))
	drc.SetRegistries(regs)
	return drc
}
