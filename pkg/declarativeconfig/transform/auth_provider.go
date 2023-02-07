package transform

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/iap"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

var _ Transformer = (*authProviderTransform)(nil)

type authProviderTransform struct{}

func newAuthProviderTransformer() *authProviderTransform {
	return &authProviderTransform{}
}

func (a *authProviderTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]proto.Message, error) {
	authProviderConfig, ok := configuration.(*declarativeconfig.AuthProvider)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for auth provider: %T", configuration)
	}
	authProviderProto := &storage.AuthProvider{
		Id:                 declarativeconfig.NewDeclarativeAuthProviderUUID(authProviderConfig.Name).String(),
		Name:               authProviderConfig.Name,
		Type:               getType(authProviderConfig),
		UiEndpoint:         authProviderConfig.UIEndpoint,
		Enabled:            true,
		Config:             getConfig(authProviderConfig),
		LoginUrl:           "/sso/login/" + declarativeconfig.NewDeclarativeAuthProviderUUID(authProviderConfig.Name).String(),
		Active:             true,
		RequiredAttributes: getRequiredAttributes(authProviderConfig.RequiredAttributes),
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
		ClaimMappings: getClaimMappings(authProviderConfig.ClaimMappings),
	}
	return map[reflect.Type][]proto.Message{
		reflect.TypeOf((*storage.AuthProvider)(nil)): {authProviderProto},
		reflect.TypeOf((*storage.Group)(nil)):        getGroups(authProviderProto.Id, authProviderConfig),
	}, nil
}

func getType(authProviderConfig *declarativeconfig.AuthProvider) string {
	switch {
	case authProviderConfig.OIDCConfig != nil:
		return oidc.TypeName
	case authProviderConfig.IAPConfig != nil:
		return iap.TypeName
	case authProviderConfig.SAMLConfig != nil:
		return saml.TypeName
	case authProviderConfig.UserpkiConfig != nil:
		return userpki.TypeName
	case authProviderConfig.OpenshiftConfig != nil && authProviderConfig.OpenshiftConfig.Enable:
		return openshift.TypeName
	default:
		return ""
	}
}

func getConfig(authProviderConfig *declarativeconfig.AuthProvider) map[string]string {
	switch getType(authProviderConfig) {
	case oidc.TypeName:
		return map[string]string{
			oidc.IssuerConfigKey:                    authProviderConfig.OIDCConfig.Issuer,
			oidc.ModeConfigKey:                      authProviderConfig.OIDCConfig.CallbackMode,
			oidc.ClientIDConfigKey:                  authProviderConfig.OIDCConfig.ClientID,
			oidc.ClientSecretConfigKey:              authProviderConfig.OIDCConfig.ClientSecret,
			oidc.DisableOfflineAccessScopeConfigKey: strconv.FormatBool(authProviderConfig.OIDCConfig.DisableOfflineAccessScope),
		}
	case iap.TypeName:
		return map[string]string{
			iap.AudienceConfigKey: authProviderConfig.IAPConfig.Audience,
		}
	case saml.TypeName:
		return map[string]string{
			saml.SpIssuerConfigKey:        authProviderConfig.SAMLConfig.SpIssuer,
			saml.IDPMetadataURLConfigKey:  authProviderConfig.SAMLConfig.MetadataURL,
			saml.IDPSSOUrlConfigKey:       authProviderConfig.SAMLConfig.SsoURL,
			saml.IDPIssuerConfigKey:       authProviderConfig.SAMLConfig.IDPIssuer,
			saml.IDPCertPemConfigKey:      authProviderConfig.SAMLConfig.Cert,
			saml.IDPNameIDFormatConfigKey: authProviderConfig.SAMLConfig.NameIDFormat,
		}
	case userpki.TypeName:
		return map[string]string{
			userpki.ConfigKeys: authProviderConfig.UserpkiConfig.CertificateAuthorities,
		}
	case openshift.TypeName:
		return map[string]string{}
	default:
		return nil
	}
}

func getRequiredAttributes(requiredAttributesConfig []declarativeconfig.RequiredAttribute) []*storage.AuthProvider_RequiredAttribute {
	requiredAttributes := make([]*storage.AuthProvider_RequiredAttribute, 0, len(requiredAttributesConfig))
	for _, req := range requiredAttributesConfig {
		requiredAttributes = append(requiredAttributes, &storage.AuthProvider_RequiredAttribute{
			AttributeKey:   req.AttributeKey,
			AttributeValue: req.AttributeValue,
		})
	}
	return requiredAttributes
}

func getClaimMappings(claimMappingsConfig []declarativeconfig.ClaimMapping) map[string]string {
	claimMappings := make(map[string]string, len(claimMappingsConfig))
	for _, mapping := range claimMappingsConfig {
		claimMappings[mapping.Name] = mapping.Path
	}
	return claimMappings
}

func getGroups(authProviderID string, authProviderConfig *declarativeconfig.AuthProvider) []proto.Message {
	groups := make([]proto.Message, 0, len(authProviderConfig.Groups)+1)

	groups = append(groups, &storage.Group{
		Props: &storage.GroupProperties{
			Id:             declarativeconfig.NewDeclarativeGroupUUID(authProviderConfig.Name + "default").String(),
			Traits:         &storage.Traits{Origin: storage.Traits_DECLARATIVE},
			AuthProviderId: authProviderID,
			Key:            "",
			Value:          "",
		},
		RoleName: authProviderConfig.MinimumRoleName,
	})

	for id, group := range authProviderConfig.Groups {
		groups = append(groups, &storage.Group{
			Props: &storage.GroupProperties{
				Id:             declarativeconfig.NewDeclarativeGroupUUID(fmt.Sprintf("%s%d", authProviderConfig.Name, id)).String(),
				Traits:         &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				AuthProviderId: authProviderID,
				Key:            group.AttributeKey,
				Value:          group.AttributeValue,
			},
			RoleName: group.RoleName,
		})
	}

	return groups
}
