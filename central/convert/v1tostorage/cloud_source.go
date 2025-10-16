package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// CloudSource converts the given v1.CloudSource to a storage.CloudSource.
func CloudSource(cloudSource *v1.CloudSource) *storage.CloudSource {
	storageCloudSource := &storage.CloudSource{}
	storageCloudSource.SetId(cloudSource.GetId())
	storageCloudSource.SetName(cloudSource.GetName())
	storageCloudSource.SetType(cloudSourceToStorageTypeEnum(cloudSource.GetType()))
	storageCloudSource.SetCredentials(cloudSourceToStorageCredentials(cloudSource.GetCredentials()))
	storageCloudSource.SetSkipTestIntegration(cloudSource.GetSkipTestIntegration())

	switch cloudSource.WhichConfig() {
	case v1.CloudSource_PaladinCloud_case:
		pcc := &storage.PaladinCloudConfig{}
		pcc.SetEndpoint(cloudSource.GetPaladinCloud().GetEndpoint())
		storageCloudSource.SetPaladinCloud(proto.ValueOrDefault(pcc))
	case v1.CloudSource_Ocm_case:
		oCMConfig := &storage.OCMConfig{}
		oCMConfig.SetEndpoint(cloudSource.GetOcm().GetEndpoint())
		storageCloudSource.SetOcm(proto.ValueOrDefault(oCMConfig))
	}
	return storageCloudSource
}

func cloudSourceToStorageTypeEnum(val v1.CloudSource_Type) storage.CloudSource_Type {
	return storage.CloudSource_Type(storage.CloudSource_Type_value[val.String()])
}

func cloudSourceToStorageCredentials(credentials *v1.CloudSource_Credentials) *storage.CloudSource_Credentials {
	cc := &storage.CloudSource_Credentials{}
	cc.SetSecret(credentials.GetSecret())
	cc.SetClientId(credentials.GetClientId())
	cc.SetClientSecret(credentials.GetClientSecret())
	return cc
}
