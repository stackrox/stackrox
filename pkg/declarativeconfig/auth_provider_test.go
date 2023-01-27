package declarativeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestAuthProviderYAMLTransformation(t *testing.T) {
	data := []byte(`
name: test-name
requiredAttributes:
- key: "groups"
  value: "stackrox"
minimumRole: "None"
uiEndpoint: "https://localhost:8000"
extraUIEndpoints: ["https://localhost:8001"]
groups:
- key: "email"
  value: "admin@stackrox.com"
  role: "Admin"
- key: "email"
  value: "someone@stackrox.com"
  role: "Analyst"
oidc:
  issuer: "https://stackrox.com"
  mode: "auto select"
  clientID: "some-client-id"
  clientSecret: "some-client-secret"
  disableOfflineAccessScope: true
`)
	ap := AuthProvider{}

	err := yaml.Unmarshal(data, &ap)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", ap.Name)
	assert.Equal(t, "None", ap.MinimumRoleName)
	assert.Equal(t, "https://localhost:8000", ap.UIEndpoint)

	assert.Len(t, ap.ExtraUIEndpoints, 1)
	assert.Equal(t, "https://localhost:8001", ap.ExtraUIEndpoints[0])

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
	assert.Equal(t, "auto select", ap.OIDCConfig.CallbackMode)
	assert.Equal(t, "https://stackrox.com", ap.OIDCConfig.Issuer)
	assert.True(t, ap.OIDCConfig.DisableOfflineAccessScope)
}
