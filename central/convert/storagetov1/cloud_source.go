package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secrets"
	"google.golang.org/protobuf/proto"
)

// CloudSource converts the given storage.CloudSource to a v1.CloudSource
// and scrubs secret fields.
func CloudSource(cloudSource *storage.CloudSource) *v1.CloudSource {
	v1CloudSource := &v1.CloudSource{}
	v1CloudSource.SetId(cloudSource.GetId())
	v1CloudSource.SetName(cloudSource.GetName())
	v1CloudSource.SetType(cloudSourceToV1TypeEnum(cloudSource.GetType()))
	v1CloudSource.SetCredentials(cloudSourceToV1Credentials(cloudSource.GetCredentials()))
	v1CloudSource.SetSkipTestIntegration(cloudSource.GetSkipTestIntegration())

	switch cloudSource.WhichConfig() {
	case storage.CloudSource_PaladinCloud_case:
		pcc := &v1.PaladinCloudConfig{}
		pcc.SetEndpoint(cloudSource.GetPaladinCloud().GetEndpoint())
		v1CloudSource.SetPaladinCloud(proto.ValueOrDefault(pcc))
	case storage.CloudSource_Ocm_case:
		oCMConfig := &v1.OCMConfig{}
		oCMConfig.SetEndpoint(cloudSource.GetOcm().GetEndpoint())
		v1CloudSource.SetOcm(proto.ValueOrDefault(oCMConfig))
	}
	secrets.ScrubSecretsFromStructWithReplacement(v1CloudSource, secrets.ScrubReplacementStr, secrets.WithScrubZeroValues(false))
	return v1CloudSource
}

func cloudSourceToV1TypeEnum(val storage.CloudSource_Type) v1.CloudSource_Type {
	return v1.CloudSource_Type(v1.CloudSource_Type_value[val.String()])
}

func cloudSourceToV1Credentials(credentials *storage.CloudSource_Credentials) *v1.CloudSource_Credentials {
	cc := &v1.CloudSource_Credentials{}
	cc.SetSecret(credentials.GetSecret())
	cc.SetClientId(credentials.GetClientId())
	cc.SetClientSecret(credentials.GetClientSecret())
	return cc
}
