package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CloudSource converts the given storage.CloudSource to a v1.CloudSource.
func CloudSource(cloudSource *storage.CloudSource) *v1.CloudSource {
	v1CloudSource := &v1.CloudSource{
		Id:                  cloudSource.GetId(),
		Name:                cloudSource.GetName(),
		Type:                cloudSourceToV1TypeEnum(cloudSource.GetType()),
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
	return v1CloudSource
}

func cloudSourceToV1TypeEnum(val storage.CloudSource_Type) v1.CloudSource_Type {
	return v1.CloudSource_Type(v1.CloudSource_Type_value[val.String()])
}
