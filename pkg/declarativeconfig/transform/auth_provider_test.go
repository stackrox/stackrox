package transform

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/iap"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformAuthProvider(t *testing.T) {
	at := newAuthProviderTransformer()
	msgs, err := at.Transform(&declarativeconfig.AccessScope{})
	assert.Nil(t, msgs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestGetAuthProviderType(t *testing.T) {
	cases := map[string]struct {
		cfg *declarativeconfig.AuthProvider
		typ string
		err error
	}{
		"oidc != nil -> oidc type": {
			cfg: &declarativeconfig.AuthProvider{OIDCConfig: &declarativeconfig.OIDCConfig{}},
			typ: oidc.TypeName,
		},
		"iap != nil -> iap type": {
			cfg: &declarativeconfig.AuthProvider{IAPConfig: &declarativeconfig.IAPConfig{}},
			typ: iap.TypeName,
		},
		"saml != nil -> saml type": {
			cfg: &declarativeconfig.AuthProvider{SAMLConfig: &declarativeconfig.SAMLConfig{}},
			typ: saml.TypeName,
		},
		"userpki != nil -> userpki type": {
			cfg: &declarativeconfig.AuthProvider{UserpkiConfig: &declarativeconfig.UserpkiConfig{}},
			typ: userpki.TypeName,
		},
		"openshift != nil && enabled -> openshift type": {
			cfg: &declarativeconfig.AuthProvider{OpenshiftConfig: &declarativeconfig.OpenshiftConfig{Enable: true}},
			typ: openshift.TypeName,
		},
		"openshift != nil && !enabled -> empty type": {
			cfg: &declarativeconfig.AuthProvider{OpenshiftConfig: &declarativeconfig.OpenshiftConfig{}},
			err: errox.InvalidArgs,
		},
		"no type set -> empty type": {
			cfg: &declarativeconfig.AuthProvider{},
			err: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			typ, err := getAuthProviderType(c.cfg)
			if c.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, c.err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.typ, typ)
		})
	}
}

func TestAuthProviderConfig(t *testing.T) {
	cases := map[string]struct {
		authProvider *declarativeconfig.AuthProvider
		cfg          map[string]string
	}{
		"no config set -> nil map": {
			authProvider: &declarativeconfig.AuthProvider{},
		},
		"openshift -> empty map": {
			authProvider: &declarativeconfig.AuthProvider{OpenshiftConfig: &declarativeconfig.OpenshiftConfig{Enable: true}},
			cfg:          map[string]string{},
		},
		"userpki config": {
			authProvider: &declarativeconfig.AuthProvider{
				UserpkiConfig: &declarativeconfig.UserpkiConfig{CertificateAuthorities: "some-value"},
			},
			cfg: map[string]string{userpki.ConfigKeys: "some-value"},
		},
		"saml config with metadata url": {
			authProvider: &declarativeconfig.AuthProvider{
				SAMLConfig: &declarativeconfig.SAMLConfig{
					SpIssuer:    "some-issuer",
					MetadataURL: "some-metadata-url",
				},
			},
			cfg: map[string]string{
				saml.SpIssuerConfigKey:        "some-issuer",
				saml.IDPMetadataURLConfigKey:  "some-metadata-url",
				saml.IDPSSOUrlConfigKey:       "",
				saml.IDPIssuerConfigKey:       "",
				saml.IDPCertPemConfigKey:      "",
				saml.IDPNameIDFormatConfigKey: "",
			},
		},
		"saml config without metadata url": {
			authProvider: &declarativeconfig.AuthProvider{
				SAMLConfig: &declarativeconfig.SAMLConfig{
					SpIssuer:     "some-issuer",
					Cert:         "some-cert",
					SsoURL:       "some-sso-url",
					NameIDFormat: "some-format",
					IDPIssuer:    "some-idp-issuer",
				},
			},
			cfg: map[string]string{
				saml.SpIssuerConfigKey:        "some-issuer",
				saml.IDPMetadataURLConfigKey:  "",
				saml.IDPSSOUrlConfigKey:       "some-sso-url",
				saml.IDPIssuerConfigKey:       "some-idp-issuer",
				saml.IDPCertPemConfigKey:      "some-cert",
				saml.IDPNameIDFormatConfigKey: "some-format",
			},
		},
		"iap config": {
			authProvider: &declarativeconfig.AuthProvider{
				IAPConfig: &declarativeconfig.IAPConfig{Audience: "some-audience"},
			},
			cfg: map[string]string{
				iap.AudienceConfigKey: "some-audience",
			},
		},
		"oidc config": {
			authProvider: &declarativeconfig.AuthProvider{
				OIDCConfig: &declarativeconfig.OIDCConfig{
					Issuer:                    "some-issuer",
					CallbackMode:              "auto",
					ClientID:                  "some-client-id",
					ClientSecret:              "some-client-secret",
					DisableOfflineAccessScope: true,
				},
			},
			cfg: map[string]string{
				oidc.IssuerConfigKey:                    "some-issuer",
				oidc.ModeConfigKey:                      "auto",
				oidc.ClientIDConfigKey:                  "some-client-id",
				oidc.ClientSecretConfigKey:              "some-client-secret",
				oidc.DisableOfflineAccessScopeConfigKey: "true",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// Error handling is tested within TestGetType.
			cfg, _ := getConfig(c.authProvider)
			assert.Equal(t, c.cfg, cfg)
		})
	}
}

func TestTransformAuthProvider(t *testing.T) {
	// Set everything and the kitchen sink:
	// - OIDC with all values given.
	// - multiple required attributes.
	// - multiple claim mappings.
	// - multiple groups.
	authProvider := &declarativeconfig.AuthProvider{
		Name:             "test-auth-provider",
		MinimumRoleName:  "Analyst",
		UIEndpoint:       "localhost:8000",
		ExtraUIEndpoints: []string{"localhost:8080", "127.0.0.1:8080"},
		Groups: []declarativeconfig.Group{
			{
				AttributeKey:   "email",
				AttributeValue: "someone@something.com",
				RoleName:       "Admin",
			},
			{
				AttributeKey:   "email",
				AttributeValue: "somewhere@something.com",
				RoleName:       "Scope Manager",
			},
			{
				AttributeKey:   "userid",
				AttributeValue: "12333",
				RoleName:       "Continous Integration",
			},
		},
		RequiredAttributes: []declarativeconfig.RequiredAttribute{
			{
				AttributeKey:   "orgid",
				AttributeValue: "12345",
			},
			{
				AttributeKey:   "custom_thing",
				AttributeValue: "some-company",
			},
		},
		ClaimMappings: []declarativeconfig.ClaimMapping{
			{
				Path: "some.nested.claim",
				Name: "custom_thing",
			},
			{
				Path: "another.one",
				Name: "another_thing",
			},
		},
		OIDCConfig: &declarativeconfig.OIDCConfig{
			Issuer:                    "http://some-issuer",
			CallbackMode:              "auto",
			ClientID:                  "some-client-id",
			ClientSecret:              "some-client-secret",
			DisableOfflineAccessScope: true,
		},
	}
	expectedAuthProviderID := declarativeconfig.NewDeclarativeAuthProviderUUID(authProvider.Name).String()
	expectedConfig := map[string]string{
		oidc.IssuerConfigKey:                    authProvider.OIDCConfig.Issuer,
		oidc.ModeConfigKey:                      authProvider.OIDCConfig.CallbackMode,
		oidc.ClientIDConfigKey:                  authProvider.OIDCConfig.ClientID,
		oidc.ClientSecretConfigKey:              authProvider.OIDCConfig.ClientSecret,
		oidc.DisableOfflineAccessScopeConfigKey: "true",
	}
	expectedClaimMappings := map[string]string{
		authProvider.ClaimMappings[0].Path: authProvider.ClaimMappings[0].Name,
		authProvider.ClaimMappings[1].Path: authProvider.ClaimMappings[1].Name,
	}
	expectedRequiredAttributes := []*storage.AuthProvider_RequiredAttribute{
		{
			AttributeKey:   authProvider.RequiredAttributes[0].AttributeKey,
			AttributeValue: authProvider.RequiredAttributes[0].AttributeValue,
		},
		{
			AttributeKey:   authProvider.RequiredAttributes[1].AttributeKey,
			AttributeValue: authProvider.RequiredAttributes[1].AttributeValue,
		},
	}

	transformer := newAuthProviderTransformer()
	protos, err := transformer.Transform(authProvider)
	assert.NoError(t, err)

	require.Contains(t, protos, authProviderType)
	require.Len(t, protos[authProviderType], 1)
	authProviderProto, ok := protos[authProviderType][0].(*storage.AuthProvider)
	require.True(t, ok)

	assert.Equal(t, storage.Traits_DECLARATIVE, authProviderProto.GetTraits().GetOrigin())

	assert.Equal(t, expectedAuthProviderID, authProviderProto.GetId())
	assert.Equal(t, authProvider.Name, authProviderProto.GetName())

	assert.Equal(t, authProvider.UIEndpoint, authProviderProto.GetUiEndpoint())
	assert.ElementsMatch(t, authProvider.ExtraUIEndpoints, authProviderProto.GetExtraUiEndpoints())

	assert.Empty(t, authProviderProto.GetLoginUrl())

	assert.True(t, authProviderProto.GetEnabled())
	assert.True(t, authProviderProto.GetActive())

	assert.Equal(t, oidc.TypeName, authProviderProto.GetType())
	assert.Equal(t, expectedConfig, authProviderProto.GetConfig())

	assert.Equal(t, expectedClaimMappings, authProviderProto.GetClaimMappings())

	assert.ElementsMatch(t, expectedRequiredAttributes, authProviderProto.GetRequiredAttributes())

	require.Contains(t, protos, groupType)
	require.Len(t, protos[groupType], 4)
	groupsProto := protos[groupType]

	defaultGroupProto := groupsProto[0]
	defaultGroup, ok := defaultGroupProto.(*storage.Group)
	require.True(t, ok)
	assert.Equal(t, declarativeconfig.NewDeclarativeGroupUUID(authProvider.Name+"-default").String(),
		defaultGroup.GetProps().GetId())
	assert.Equal(t, authProvider.MinimumRoleName, defaultGroup.GetRoleName())
	assert.Equal(t, expectedAuthProviderID, defaultGroup.GetProps().GetAuthProviderId())
	assert.Empty(t, defaultGroup.GetProps().GetKey())
	assert.Empty(t, defaultGroup.GetProps().GetValue())

	groupsProto = groupsProto[1:]

	for id, groupProto := range groupsProto {
		group, ok := groupProto.(*storage.Group)
		require.True(t, ok)

		assert.Equal(t, declarativeconfig.NewDeclarativeGroupUUID(fmt.Sprintf("%s-%d", authProvider.Name, id)).String(),
			group.GetProps().GetId())
		assert.Equal(t, authProvider.Groups[id].RoleName, group.GetRoleName())
		assert.Equal(t, expectedAuthProviderID, group.GetProps().GetAuthProviderId())
		assert.Equal(t, authProvider.Groups[id].AttributeKey, group.GetProps().GetKey())
		assert.Equal(t, authProvider.Groups[id].AttributeValue, group.GetProps().GetValue())
	}
}

func TestTransformAuthProvider_NoMinimumRoleName(t *testing.T) {
	authProvider := &declarativeconfig.AuthProvider{
		Name:       "test-auth-provider",
		UIEndpoint: "localhost:8000",
		OIDCConfig: &declarativeconfig.OIDCConfig{
			Issuer:       "http://some-issuer",
			CallbackMode: "auto",
			ClientID:     "some-client-id",
			ClientSecret: "some-client-secret",
		},
	}

	transformer := newAuthProviderTransformer()
	protos, err := transformer.Transform(authProvider)
	assert.NoError(t, err)

	require.Contains(t, protos, authProviderType)
	require.Len(t, protos[authProviderType], 1)
	_, ok := protos[authProviderType][0].(*storage.AuthProvider)
	require.True(t, ok)

	require.Contains(t, protos, groupType)
	require.Len(t, protos[groupType], 0)
}
