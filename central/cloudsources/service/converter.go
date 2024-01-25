package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secrets"
)

func toV1Proto(cloudSource *storage.CloudSource) *v1.CloudSource {
	v1CloudSource := &v1.CloudSource{
		Id:                  cloudSource.GetId(),
		Name:                cloudSource.GetName(),
		Type:                toV1TypeEnum(cloudSource.GetType()),
		SkipTestIntegration: cloudSource.GetSkipTestIntegration(),
	}

	switch config := cloudSource.GetConfig().(type) {
	case *storage.CloudSource_PaladinCloud:
		v1CloudSource.Config = &v1.CloudSource_PaladinCloud{
			PaladinCloud: &v1.PaladinCloudConfig{
				Endpoint: config.PaladinCloud.GetEndpoint(),
			},
		}
	case *storage.CloudSource_Ocm:
		v1CloudSource.Config = &v1.CloudSource_Ocm{
			Ocm: &v1.OCMConfig{
				Endpoint: config.Ocm.GetEndpoint(),
			},
		}
	}
	// Credentials are left out by the storage -> api layer conversion.
	// We scrub here just in case for additional redundancy.
	secrets.ScrubSecretsFromStructWithReplacement(v1CloudSource, secrets.ScrubReplacementStr)
	return v1CloudSource
}

func toStorageProto(cloudSource *v1.CloudSource) *storage.CloudSource {
	storageCloudSource := &storage.CloudSource{
		Id:                  cloudSource.GetId(),
		Name:                cloudSource.GetName(),
		Type:                toStorageTypeEnum(cloudSource.GetType()),
		Credentials:         toStorageCredentials(cloudSource.GetCredentials()),
		SkipTestIntegration: cloudSource.GetSkipTestIntegration(),
	}

	switch config := cloudSource.GetConfig().(type) {
	case *v1.CloudSource_PaladinCloud:
		storageCloudSource.Config = &storage.CloudSource_PaladinCloud{
			PaladinCloud: &storage.PaladinCloudConfig{
				Endpoint: config.PaladinCloud.GetEndpoint(),
			},
		}
	case *v1.CloudSource_Ocm:
		storageCloudSource.Config = &storage.CloudSource_Ocm{
			Ocm: &storage.OCMConfig{
				Endpoint: config.Ocm.GetEndpoint(),
			},
		}
	}
	return storageCloudSource
}

func toV1TypeEnum(val storage.CloudSource_Type) v1.CloudSource_Type {
	return v1.CloudSource_Type(v1.CloudSource_Type_value[val.String()])
}

func toStorageTypeEnum(val v1.CloudSource_Type) storage.CloudSource_Type {
	return storage.CloudSource_Type(storage.CloudSource_Type_value[val.String()])
}

func toStorageCredentials(credentials *v1.CloudSource_Credentials) *storage.CloudSource_Credentials {
	return &storage.CloudSource_Credentials{
		Secret: credentials.GetSecret(),
	}
}
