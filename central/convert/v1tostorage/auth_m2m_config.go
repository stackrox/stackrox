package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AuthM2MConfig converts the given v1.AuthMachineToMachineConfig to storage.AuthMachineToMachineConfig.
func AuthM2MConfig(config *v1.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	storageConfig := &storage.AuthMachineToMachineConfig{
		Id:                      config.GetId(),
		Type:                    convertTypeEnum(config.GetType()),
		TokenExpirationDuration: config.GetTokenExpirationDuration(),
		Mappings:                convertMappings(config.GetMappings()),
		Issuer:                  setIssuer(config.GetType(), config.GetIssuer()),
	}

	return storageConfig
}

func convertMappings(mappings []*v1.AuthMachineToMachineConfig_Mapping) []*storage.AuthMachineToMachineConfig_Mapping {
	if len(mappings) == 0 {
		return nil
	}
	storageMappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		storageMappings = append(storageMappings, &storage.AuthMachineToMachineConfig_Mapping{
			Key:             mapping.GetKey(),
			ValueExpression: mapping.GetValueExpression(),
			Role:            mapping.GetRole(),
		})
	}
	return storageMappings
}

func setIssuer(typ v1.AuthMachineToMachineConfig_Type, issuer string) string {
	switch typ {
	case v1.AuthMachineToMachineConfig_GITHUB_ACTIONS:
		return "https://token.actions.githubusercontent.com"
	default:
		return issuer
	}
}

func convertTypeEnum(val v1.AuthMachineToMachineConfig_Type) storage.AuthMachineToMachineConfig_Type {
	return storage.AuthMachineToMachineConfig_Type(storage.AuthMachineToMachineConfig_Type_value[val.String()])
}
