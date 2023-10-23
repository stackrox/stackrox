package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AuthM2MConfigs converts the given list of storage.AuthMachineToMachineConfig to a list of
// v1.AuthMachineToMachineConfig.
func AuthM2MConfigs(configs []*storage.AuthMachineToMachineConfig) []*v1.AuthMachineToMachineConfig {
	v1Configs := make([]*v1.AuthMachineToMachineConfig, 0, len(configs))
	for _, config := range configs {
		v1Configs = append(v1Configs, AuthM2MConfig(config))
	}
	return v1Configs
}

// AuthM2MConfig converts the given storage.AuthMachineToMachineConfig to v1.AuthMachineToMachineConfig.
func AuthM2MConfig(config *storage.AuthMachineToMachineConfig) *v1.AuthMachineToMachineConfig {
	v1Config := &v1.AuthMachineToMachineConfig{
		Id:                      config.GetId(),
		Type:                    convertTypeEnum(config.GetType()),
		TokenExpirationDuration: config.GetTokenExpirationDuration(),
		Mappings:                convertMappings(config.GetMappings()),
		Issuer:                  config.GetIssuer(),
	}

	return v1Config
}

func convertTypeEnum(val storage.AuthMachineToMachineConfig_Type) v1.AuthMachineToMachineConfig_Type {
	return v1.AuthMachineToMachineConfig_Type(v1.AuthMachineToMachineConfig_Type_value[val.String()])
}

func convertMappings(mappings []*storage.AuthMachineToMachineConfig_Mapping) []*v1.AuthMachineToMachineConfig_Mapping {
	if len(mappings) == 0 {
		return nil
	}
	v1Mappings := make([]*v1.AuthMachineToMachineConfig_Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		v1Mappings = append(v1Mappings, &v1.AuthMachineToMachineConfig_Mapping{
			Key:   mapping.GetKey(),
			Value: mapping.GetValue(),
			Role:  mapping.GetRole(),
		})
	}
	return v1Mappings
}
