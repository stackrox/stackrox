package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func toV1Protos(configs []*storage.AuthMachineToMachineConfig) []*v1.AuthMachineToMachineConfig {
	v1Configs := make([]*v1.AuthMachineToMachineConfig, 0, len(configs))
	for _, config := range configs {
		v1Configs = append(v1Configs, toV1Proto(config))
	}
	return v1Configs
}

func toV1Proto(config *storage.AuthMachineToMachineConfig) *v1.AuthMachineToMachineConfig {
	return &v1.AuthMachineToMachineConfig{
		Id:                      config.GetId(),
		Type:                    toV1TypeEnum(config.GetType()),
		TokenExpirationDuration: config.GetTokenExpirationDuration(),
		Mappings:                toV1Mappings(config.GetMappings()),
		IssuerConfig:            toV1IssuerConfig(config.GetGeneric()),
	}
}

func toStorageProto(config *v1.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	return &storage.AuthMachineToMachineConfig{
		Id:                      config.GetId(),
		Type:                    toStorageTypeEnum(config.GetType()),
		TokenExpirationDuration: config.GetTokenExpirationDuration(),
		Mappings:                toStorageMappings(config.GetMappings()),
		IssuerConfig:            toStorageIssuerConfig(config.GetGeneric()),
	}
}

func toV1IssuerConfig(config *storage.AuthMachineToMachineConfig_GenericIssuer) *v1.AuthMachineToMachineConfig_Generic {
	if config == nil {
		return nil
	}
	return &v1.AuthMachineToMachineConfig_Generic{Generic: &v1.AuthMachineToMachineConfig_GenericIssuer{
		Issuer: config.GetIssuer(),
	}}
}

func toV1TypeEnum(val storage.AuthMachineToMachineConfig_Type) v1.AuthMachineToMachineConfig_Type {
	return v1.AuthMachineToMachineConfig_Type(v1.AuthMachineToMachineConfig_Type_value[val.String()])
}

func toV1Mappings(mappings []*storage.AuthMachineToMachineConfig_Mapping) []*v1.AuthMachineToMachineConfig_Mapping {
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

func toStorageIssuerConfig(config *v1.AuthMachineToMachineConfig_GenericIssuer) *storage.AuthMachineToMachineConfig_Generic {
	if config == nil {
		return nil
	}
	return &storage.AuthMachineToMachineConfig_Generic{Generic: &storage.AuthMachineToMachineConfig_GenericIssuer{
		Issuer: config.GetIssuer(),
	}}
}

func toStorageMappings(mappings []*v1.AuthMachineToMachineConfig_Mapping) []*storage.AuthMachineToMachineConfig_Mapping {
	storageMappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		storageMappings = append(storageMappings, &storage.AuthMachineToMachineConfig_Mapping{
			Key:   mapping.GetKey(),
			Value: mapping.GetValue(),
			Role:  mapping.GetRole(),
		})
	}
	return storageMappings
}

func toStorageTypeEnum(val v1.AuthMachineToMachineConfig_Type) storage.AuthMachineToMachineConfig_Type {
	return storage.AuthMachineToMachineConfig_Type(storage.AuthMachineToMachineConfig_Type_value[val.String()])
}
