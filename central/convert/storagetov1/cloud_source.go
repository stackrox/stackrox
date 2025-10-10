package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secrets"
)

// CloudSource converts the given storage.CloudSource to a v1.CloudSource
// and scrubs secret fields.
func CloudSource(cloudSource *storage.CloudSource) *v1.CloudSource {
	id := cloudSource.GetId()
	name := cloudSource.GetName()
	typeEnum := cloudSourceToV1TypeEnum(cloudSource.GetType())
	skipTestIntegration := cloudSource.GetSkipTestIntegration()
	v1CloudSource := v1.CloudSource_builder{
		Id:                  &id,
		Name:                &name,
		Type:                &typeEnum,
		Credentials:         cloudSourceToV1Credentials(cloudSource.GetCredentials()),
		SkipTestIntegration: &skipTestIntegration,
	}.Build()

	switch config := cloudSource.GetConfig().(type) {
	case *storage.CloudSource_PaladinCloud:
		endpoint := config.PaladinCloud.GetEndpoint()
		v1CloudSource.Config = &v1.CloudSource_PaladinCloud{
			PaladinCloud: v1.PaladinCloudConfig_builder{
				Endpoint: &endpoint,
			}.Build(),
		}
	case *storage.CloudSource_Ocm:
		endpoint := config.Ocm.GetEndpoint()
		v1CloudSource.Config = &v1.CloudSource_Ocm{
			Ocm: v1.OCMConfig_builder{
				Endpoint: &endpoint,
			}.Build(),
		}
	}
	secrets.ScrubSecretsFromStructWithReplacement(v1CloudSource, secrets.ScrubReplacementStr, secrets.WithScrubZeroValues(false))
	return v1CloudSource
}

func cloudSourceToV1TypeEnum(val storage.CloudSource_Type) v1.CloudSource_Type {
	return v1.CloudSource_Type(v1.CloudSource_Type_value[val.String()])
}

func cloudSourceToV1Credentials(credentials *storage.CloudSource_Credentials) *v1.CloudSource_Credentials {
	secret := credentials.GetSecret()
	clientId := credentials.GetClientId()
	clientSecret := credentials.GetClientSecret()
	return v1.CloudSource_Credentials_builder{
		Secret:       &secret,
		ClientId:     &clientId,
		ClientSecret: &clientSecret,
	}.Build()
}
