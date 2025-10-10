package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CloudSource converts the given v1.CloudSource to a storage.CloudSource.
func CloudSource(cloudSource *v1.CloudSource) *storage.CloudSource {
	id := cloudSource.GetId()
	name := cloudSource.GetName()
	typeEnum := cloudSourceToStorageTypeEnum(cloudSource.GetType())
	skipTestIntegration := cloudSource.GetSkipTestIntegration()
	storageCloudSource := storage.CloudSource_builder{
		Id:                  &id,
		Name:                &name,
		Type:                &typeEnum,
		Credentials:         cloudSourceToStorageCredentials(cloudSource.GetCredentials()),
		SkipTestIntegration: &skipTestIntegration,
	}.Build()

	switch config := cloudSource.GetConfig().(type) {
	case *v1.CloudSource_PaladinCloud:
		endpoint := config.PaladinCloud.GetEndpoint()
		storageCloudSource.Config = &storage.CloudSource_PaladinCloud{
			PaladinCloud: storage.PaladinCloudConfig_builder{
				Endpoint: &endpoint,
			}.Build(),
		}
	case *v1.CloudSource_Ocm:
		endpoint := config.Ocm.GetEndpoint()
		storageCloudSource.Config = &storage.CloudSource_Ocm{
			Ocm: storage.OCMConfig_builder{
				Endpoint: &endpoint,
			}.Build(),
		}
	}
	return storageCloudSource
}

func cloudSourceToStorageTypeEnum(val v1.CloudSource_Type) storage.CloudSource_Type {
	return storage.CloudSource_Type(storage.CloudSource_Type_value[val.String()])
}

func cloudSourceToStorageCredentials(credentials *v1.CloudSource_Credentials) *storage.CloudSource_Credentials {
	secret := credentials.GetSecret()
	clientId := credentials.GetClientId()
	clientSecret := credentials.GetClientSecret()
	return storage.CloudSource_Credentials_builder{
		Secret:       &secret,
		ClientId:     &clientId,
		ClientSecret: &clientSecret,
	}.Build()
}
