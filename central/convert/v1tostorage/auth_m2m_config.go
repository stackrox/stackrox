package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AuthM2MConfig converts the given v1.AuthMachineToMachineConfig to storage.AuthMachineToMachineConfig.
func AuthM2MConfig(config *v1.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	id := config.GetId()
	typeEnum := convertTypeEnum(config.GetType())
	issuer := config.GetIssuer()
	tokenExpiration := config.GetTokenExpirationDuration()
	storageConfig := storage.AuthMachineToMachineConfig_builder{
		Id:                      &id,
		Type:                    &typeEnum,
		TokenExpirationDuration: &tokenExpiration,
		Mappings:                convertMappings(config.GetMappings()),
		Issuer:                  &issuer,
		Traits:                  Traits(config.GetTraits()),
	}.Build()

	return storageConfig
}

func convertMappings(mappings []*v1.AuthMachineToMachineConfig_Mapping) []*storage.AuthMachineToMachineConfig_Mapping {
	if len(mappings) == 0 {
		return nil
	}
	storageMappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		key := mapping.GetKey()
		valueExpression := mapping.GetValueExpression()
		role := mapping.GetRole()
		storageMappings = append(storageMappings, storage.AuthMachineToMachineConfig_Mapping_builder{
			Key:             &key,
			ValueExpression: &valueExpression,
			Role:            &role,
		}.Build())
	}
	return storageMappings
}

func convertTypeEnum(val v1.AuthMachineToMachineConfig_Type) storage.AuthMachineToMachineConfig_Type {
	return storage.AuthMachineToMachineConfig_Type(storage.AuthMachineToMachineConfig_Type_value[val.String()])
}
