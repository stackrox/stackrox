package convert

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ConfigToAPI converts a delegated registry config from the type used by storage (db) to
// the type used by the REST API
func ConfigToAPI(from *storage.DelegatedRegistryConfig) *v1.DelegatedRegistryConfig {
	var regs []*v1.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*v1.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &v1.DelegatedRegistryConfig_DelegatedRegistry{
				ClusterId:    reg.ClusterId,
				RegistryPath: reg.RegistryPath,
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

// ConfigToStorage converts a delegated registry config from the type used by the REST API to
// the type used by storage (db)
func ConfigToStorage(from *v1.DelegatedRegistryConfig) *storage.DelegatedRegistryConfig {
	var regs []*storage.DelegatedRegistryConfig_DelegatedRegistry

	if len(from.Registries) > 0 {
		regs = make([]*storage.DelegatedRegistryConfig_DelegatedRegistry, len(from.Registries))

		for i, reg := range from.Registries {
			regs[i] = &storage.DelegatedRegistryConfig_DelegatedRegistry{
				ClusterId:    reg.ClusterId,
				RegistryPath: reg.RegistryPath,
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
