package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AuthM2MConfig converts the given v1.AuthMachineToMachineConfig to storage.AuthMachineToMachineConfig.
func AuthM2MConfig(config *v1.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	storageConfig := &storage.AuthMachineToMachineConfig{}
	storageConfig.SetId(config.GetId())
	storageConfig.SetType(convertTypeEnum(config.GetType()))
	storageConfig.SetTokenExpirationDuration(config.GetTokenExpirationDuration())
	storageConfig.SetMappings(convertMappings(config.GetMappings()))
	storageConfig.SetIssuer(config.GetIssuer())
	storageConfig.SetTraits(Traits(config.GetTraits()))

	return storageConfig
}

func convertMappings(mappings []*v1.AuthMachineToMachineConfig_Mapping) []*storage.AuthMachineToMachineConfig_Mapping {
	if len(mappings) == 0 {
		return nil
	}
	storageMappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		am := &storage.AuthMachineToMachineConfig_Mapping{}
		am.SetKey(mapping.GetKey())
		am.SetValueExpression(mapping.GetValueExpression())
		am.SetRole(mapping.GetRole())
		storageMappings = append(storageMappings, am)
	}
	return storageMappings
}

func convertTypeEnum(val v1.AuthMachineToMachineConfig_Type) storage.AuthMachineToMachineConfig_Type {
	return storage.AuthMachineToMachineConfig_Type(storage.AuthMachineToMachineConfig_Type_value[val.String()])
}
