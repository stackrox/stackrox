package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CloudSource converts the given v1.CloudSource to a storage.CloudSource.
func CloudSource(cloudSource *v1.CloudSource) *storage.CloudSource {
	storageCloudSource := &storage.CloudSource{
		Id:                  cloudSource.GetId(),
		Name:                cloudSource.GetName(),
		Type:                cloudSourceToStorageTypeEnum(cloudSource.GetType()),
		Credentials:         cloudSourceToStorageCredentials(cloudSource.GetCredentials()),
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

func cloudSourceToStorageTypeEnum(val v1.CloudSource_Type) storage.CloudSource_Type {
	return storage.CloudSource_Type(storage.CloudSource_Type_value[val.String()])
}

func cloudSourceToStorageCredentials(credentials *v1.CloudSource_Credentials) *storage.CloudSource_Credentials {
	return &storage.CloudSource_Credentials{
		Secret:       credentials.GetSecret(),
		ClientId:     credentials.GetClientId(),
		ClientSecret: credentials.GetClientSecret(),
	}
}
