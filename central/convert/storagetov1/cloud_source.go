package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secrets"
)

// CloudSource converts the given storage.CloudSource to a v1.CloudSource
// and scrubs secret fields.
func CloudSource(cloudSource *storage.CloudSource) *v1.CloudSource {
	v1CloudSource := &v1.CloudSource{
		Id:                  cloudSource.GetId(),
		Name:                cloudSource.GetName(),
		Type:                cloudSourceToV1TypeEnum(cloudSource.GetType()),
		Credentials:         cloudSourceToV1Credentials(cloudSource.GetCredentials()),
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
	secrets.ScrubSecretsFromStructWithReplacement(v1CloudSource, secrets.ScrubReplacementStr)
	return v1CloudSource
}

func cloudSourceToV1TypeEnum(val storage.CloudSource_Type) v1.CloudSource_Type {
	return v1.CloudSource_Type(v1.CloudSource_Type_value[val.String()])
}

func cloudSourceToV1Credentials(credentials *storage.CloudSource_Credentials) *v1.CloudSource_Credentials {
	return &v1.CloudSource_Credentials{
		Secret:       credentials.GetSecret(),
		ClientId:     credentials.GetClientId(),
		ClientSecret: credentials.GetClientSecret(),
	}
}
