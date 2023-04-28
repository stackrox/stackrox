package declarativeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestAuthProviderYAMLTransformation_OIDC(t *testing.T) {
	data := []byte(`name: test-name
minimumRole: None
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:8001
groups:
    - key: email
      value: admin@stackrox.com
      role: Admin
    - key: email
      value: someone@stackrox.com
      role: Analyst
requiredAttributes:
    - key: groups
      value: stackrox
oidc:
    issuer: https://stackrox.com
    mode: auto
    clientID: some-client-id
    clientSecret: some-client-secret
    disableOfflineAccessScope: true
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Equal(t, "None", ap.MinimumRoleName)
	assert.Equal(t, "localhost:8000", ap.UIEndpoint)

	assert.Len(t, ap.ExtraUIEndpoints, 1)
	assert.Equal(t, "localhost:8001", ap.ExtraUIEndpoints[0])

	assert.Len(t, ap.Groups, 2)
	assert.Equal(t, "email", ap.Groups[0].AttributeKey)
	assert.Equal(t, "admin@stackrox.com", ap.Groups[0].AttributeValue)
	assert.Equal(t, "Admin", ap.Groups[0].RoleName)

	assert.Equal(t, "email", ap.Groups[1].AttributeKey)
	assert.Equal(t, "someone@stackrox.com", ap.Groups[1].AttributeValue)
	assert.Equal(t, "Analyst", ap.Groups[1].RoleName)

	assert.Len(t, ap.RequiredAttributes, 1)
	assert.Equal(t, "groups", ap.RequiredAttributes[0].AttributeKey)
	assert.Equal(t, "stackrox", ap.RequiredAttributes[0].AttributeValue)

	assert.Len(t, ap.ClaimMappings, 0)
	assert.Nil(t, ap.SAMLConfig)

	assert.NotNil(t, ap.OIDCConfig)
	assert.Equal(t, "some-client-id", ap.OIDCConfig.ClientID)
	assert.Equal(t, "some-client-secret", ap.OIDCConfig.ClientSecret)
	assert.Equal(t, "auto", ap.OIDCConfig.CallbackMode)
	assert.Equal(t, "https://stackrox.com", ap.OIDCConfig.Issuer)
	assert.True(t, ap.OIDCConfig.DisableOfflineAccessScope)

	bytes, err := yaml.Marshal(&ap)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}

func TestAuthProviderYAMLTransformation_SAML(t *testing.T) {
	// 1. Configure SAML using metadata URL.
	data := []byte(`
name: test-name
saml:
  spIssuer: "https://stackrox.com"
  metadataURL: "https://stackrox.com/metadata"
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Empty(t, ap.MinimumRoleName)
	assert.Empty(t, ap.UIEndpoint)
	assert.Len(t, ap.ExtraUIEndpoints, 0)
	assert.Len(t, ap.Groups, 0)
	assert.Len(t, ap.RequiredAttributes, 0)

	assert.NotNil(t, ap.SAMLConfig)
	assert.Equal(t, "https://stackrox.com", ap.SAMLConfig.SpIssuer)
	assert.Equal(t, "https://stackrox.com/metadata", ap.SAMLConfig.MetadataURL)

	assert.Nil(t, ap.OIDCConfig)
	assert.Nil(t, ap.IAPConfig)
	assert.Nil(t, ap.OpenshiftConfig)
	assert.Nil(t, ap.UserpkiConfig)

	// 2. Configure SAML without metadata URL.
	data = []byte(`
name: test-name
saml:
  spIssuer: "https://stackrox.com"
  ssoURL: "https://auth.stackrox.com"
  cert: "cert-pem"
  nameIdFormat: "emailAddress"
`)
	ap = AuthProvider{}

	err = yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Empty(t, ap.MinimumRoleName)
	assert.Empty(t, ap.UIEndpoint)
	assert.Len(t, ap.ExtraUIEndpoints, 0)
	assert.Len(t, ap.Groups, 0)
	assert.Len(t, ap.RequiredAttributes, 0)

	assert.NotNil(t, ap.SAMLConfig)
	assert.Equal(t, "https://stackrox.com", ap.SAMLConfig.SpIssuer)
	assert.Equal(t, "https://auth.stackrox.com", ap.SAMLConfig.SsoURL)
	assert.Equal(t, "cert-pem", ap.SAMLConfig.Cert)
	assert.Equal(t, "emailAddress", ap.SAMLConfig.NameIDFormat)

	assert.Nil(t, ap.OIDCConfig)
	assert.Nil(t, ap.IAPConfig)
	assert.Nil(t, ap.OpenshiftConfig)
	assert.Nil(t, ap.UserpkiConfig)
}

func TestAuthProviderYAMLTransformation_UserCertificates(t *testing.T) {
	// 1. Configure SAML using metadata URL.
	data := []byte(`
name: test-name
userpki:
  certificateAuthorities: "certs"
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Empty(t, ap.MinimumRoleName)
	assert.Empty(t, ap.UIEndpoint)
	assert.Len(t, ap.ExtraUIEndpoints, 0)
	assert.Len(t, ap.Groups, 0)
	assert.Len(t, ap.RequiredAttributes, 0)

	assert.NotNil(t, ap.UserpkiConfig)
	assert.Equal(t, "certs", ap.UserpkiConfig.CertificateAuthorities)

	assert.Nil(t, ap.OIDCConfig)
	assert.Nil(t, ap.IAPConfig)
	assert.Nil(t, ap.OpenshiftConfig)
	assert.Nil(t, ap.SAMLConfig)
}

func TestAuthProviderYAMLTransformation_IAP(t *testing.T) {
	// 1. Configure SAML using metadata URL.
	data := []byte(`
name: test-name
iap:
  audience: stackrox
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Empty(t, ap.MinimumRoleName)
	assert.Empty(t, ap.UIEndpoint)
	assert.Len(t, ap.ExtraUIEndpoints, 0)
	assert.Len(t, ap.Groups, 0)
	assert.Len(t, ap.RequiredAttributes, 0)

	assert.NotNil(t, ap.IAPConfig)
	assert.Equal(t, "stackrox", ap.IAPConfig.Audience)

	assert.Nil(t, ap.OIDCConfig)
	assert.Nil(t, ap.SAMLConfig)
	assert.Nil(t, ap.OpenshiftConfig)
	assert.Nil(t, ap.UserpkiConfig)
}

func TestAuthProviderYAMLTransformation_Openshift(t *testing.T) {
	// 1. Configure SAML using metadata URL.
	data := []byte(`
name: test-name
openshift:
  enable: true
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Empty(t, ap.MinimumRoleName)
	assert.Empty(t, ap.UIEndpoint)
	assert.Len(t, ap.ExtraUIEndpoints, 0)
	assert.Len(t, ap.Groups, 0)
	assert.Len(t, ap.RequiredAttributes, 0)

	assert.NotNil(t, ap.OpenshiftConfig)
	assert.True(t, ap.OpenshiftConfig.Enable)

	assert.Nil(t, ap.OIDCConfig)
	assert.Nil(t, ap.SAMLConfig)
	assert.Nil(t, ap.IAPConfig)
	assert.Nil(t, ap.UserpkiConfig)
}
