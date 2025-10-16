package transform

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/iap"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/utils"
)

var _ Transformer = (*authProviderTransform)(nil)

var (
	authProviderType = reflect.TypeOf((*storage.AuthProvider)(nil))
	groupType        = reflect.TypeOf((*storage.Group)(nil))
)

type authProviderTransform struct{}

func newAuthProviderTransformer() *authProviderTransform {
	return &authProviderTransform{}
}

func (a *authProviderTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
	authProviderConfig, ok := configuration.(*declarativeconfig.AuthProvider)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for auth provider: %T", configuration)
	}

	providerType, err := getAuthProviderType(authProviderConfig)
	if err != nil {
		return nil, errors.Wrap(err, "transforming auth provider type")
	}

	providerConfig, err := getConfig(authProviderConfig)
	if err != nil {
		return nil, errors.Wrap(err, "transforming auth provider configuration")
	}

	// The assumption is that, when the auth provider will be stored later on, the DefaultLoginURL option is used,
	// thus we do not set the login URL explicitly, even though we could.
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	authProviderProto := &storage.AuthProvider{}
	authProviderProto.SetId(declarativeconfig.NewDeclarativeAuthProviderUUID(authProviderConfig.Name).String())
	authProviderProto.SetName(authProviderConfig.Name)
	authProviderProto.SetType(providerType)
	authProviderProto.SetUiEndpoint(authProviderConfig.UIEndpoint)
	authProviderProto.SetExtraUiEndpoints(authProviderConfig.ExtraUIEndpoints)
	authProviderProto.SetEnabled(true) // Enabled is required to be set to ensure the auth provider listed as ready for login.
	authProviderProto.SetActive(true)  // Active signals at least one user has logged in with the auth provider and disables modification in the UI.
	authProviderProto.SetConfig(providerConfig)
	authProviderProto.SetRequiredAttributes(getRequiredAttributes(authProviderConfig.RequiredAttributes))
	authProviderProto.SetTraits(traits)
	authProviderProto.SetClaimMappings(getClaimMappings(authProviderConfig.ClaimMappings))
	return map[reflect.Type][]protocompat.Message{
		authProviderType: {authProviderProto},
		groupType:        getGroups(authProviderProto.GetId(), authProviderConfig),
	}, nil
}

func getAuthProviderType(authProviderConfig *declarativeconfig.AuthProvider) (string, error) {
	switch {
	case authProviderConfig.OIDCConfig != nil:
		return oidc.TypeName, nil
	case authProviderConfig.IAPConfig != nil:
		return iap.TypeName, nil
	case authProviderConfig.SAMLConfig != nil:
		return saml.TypeName, nil
	case authProviderConfig.UserpkiConfig != nil:
		return userpki.TypeName, nil
	case authProviderConfig.OpenshiftConfig != nil && authProviderConfig.OpenshiftConfig.Enable:
		return openshift.TypeName, nil
	default:
		return "", errox.InvalidArgs.New("no valid auth provider config given")
	}
}

func getConfig(authProviderConfig *declarativeconfig.AuthProvider) (map[string]string, error) {
	authProviderType, err := getAuthProviderType(authProviderConfig)
	if err != nil {
		return nil, err
	}
	switch authProviderType {
	case oidc.TypeName:
		return map[string]string{
			oidc.IssuerConfigKey:                    authProviderConfig.OIDCConfig.Issuer,
			oidc.ModeConfigKey:                      authProviderConfig.OIDCConfig.CallbackMode,
			oidc.ClientIDConfigKey:                  authProviderConfig.OIDCConfig.ClientID,
			oidc.ClientSecretConfigKey:              authProviderConfig.OIDCConfig.ClientSecret,
			oidc.DisableOfflineAccessScopeConfigKey: strconv.FormatBool(authProviderConfig.OIDCConfig.DisableOfflineAccessScope),
		}, nil
	case iap.TypeName:
		return map[string]string{
			iap.AudienceConfigKey: authProviderConfig.IAPConfig.Audience,
		}, nil
	case saml.TypeName:
		return map[string]string{
			saml.SpIssuerConfigKey:        authProviderConfig.SAMLConfig.SpIssuer,
			saml.IDPMetadataURLConfigKey:  authProviderConfig.SAMLConfig.MetadataURL,
			saml.IDPSSOUrlConfigKey:       authProviderConfig.SAMLConfig.SsoURL,
			saml.IDPIssuerConfigKey:       authProviderConfig.SAMLConfig.IDPIssuer,
			saml.IDPCertPemConfigKey:      authProviderConfig.SAMLConfig.Cert,
			saml.IDPNameIDFormatConfigKey: authProviderConfig.SAMLConfig.NameIDFormat,
		}, nil
	case userpki.TypeName:
		return map[string]string{
			userpki.ConfigKeys: authProviderConfig.UserpkiConfig.CertificateAuthorities,
		}, nil
	case openshift.TypeName:
		return map[string]string{}, nil
	default:
		return nil, errox.InvalidArgs.Newf("unsupported auth provider type %q given", authProviderType)
	}
}

func getRequiredAttributes(requiredAttributesConfig []declarativeconfig.RequiredAttribute) []*storage.AuthProvider_RequiredAttribute {
	requiredAttributes := make([]*storage.AuthProvider_RequiredAttribute, 0, len(requiredAttributesConfig))
	for _, req := range requiredAttributesConfig {
		ar := &storage.AuthProvider_RequiredAttribute{}
		ar.SetAttributeKey(req.AttributeKey)
		ar.SetAttributeValue(req.AttributeValue)
		requiredAttributes = append(requiredAttributes, ar)
	}
	return requiredAttributes
}

func getClaimMappings(claimMappingsConfig []declarativeconfig.ClaimMapping) map[string]string {
	claimMappings := make(map[string]string, len(claimMappingsConfig))
	for _, mapping := range claimMappingsConfig {
		claimMappings[mapping.Path] = mapping.Name
	}
	return claimMappings
}

func getGroups(authProviderID string, authProviderConfig *declarativeconfig.AuthProvider) []protocompat.Message {
	hasMinimumRoleName := authProviderConfig.MinimumRoleName != ""
	groups := make([]protocompat.Message, 0, len(authProviderConfig.Groups)+utils.IfThenElse(hasMinimumRoleName, 1, 0))

	if hasMinimumRoleName {
		traits := &storage.Traits{}
		traits.SetOrigin(storage.Traits_DECLARATIVE)
		gp := &storage.GroupProperties{}
		gp.SetId(declarativeconfig.NewDeclarativeGroupUUID(authProviderConfig.Name + "-default").String())
		gp.SetTraits(traits)
		gp.SetAuthProviderId(authProviderID)
		gp.SetKey("")
		gp.SetValue("")
		group := &storage.Group{}
		group.SetProps(gp)
		group.SetRoleName(authProviderConfig.MinimumRoleName)
		groups = append(groups, group)
	}

	for idx, group := range authProviderConfig.Groups {
		traits := &storage.Traits{}
		traits.SetOrigin(storage.Traits_DECLARATIVE)
		gp := &storage.GroupProperties{}
		gp.SetId(declarativeconfig.NewDeclarativeGroupUUID(fmt.Sprintf("%s-%d", authProviderConfig.Name, idx)).String())
		gp.SetTraits(traits)
		gp.SetAuthProviderId(authProviderID)
		gp.SetKey(group.AttributeKey)
		gp.SetValue(group.AttributeValue)
		group2 := &storage.Group{}
		group2.SetProps(gp)
		group2.SetRoleName(group.RoleName)
		groups = append(groups, group2)
	}

	return groups
}
