package declarativeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNotifierYAMLTransformation_Generic(t *testing.T) {
	data := []byte(`name: test-name
generic:
    endpoint: https://stackrox.com
    skipTLSVerify: true
    caCertPEM: stackrox-ca-cert
    username: Safest Password Generator
    password: qwerty
    headers:
        - key: Content-Length
          value: "120"
        - key: Authorization
          value: Basic
    extraFields:
        - key: extra
          value: field
    auditLoggingEnabled: true
`)
	notifier := Notifier{}

	err := yaml.Unmarshal(data, &notifier)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", notifier.Name)
	assert.NotNil(t, notifier.GenericConfig)
	assert.Equal(t, "https://stackrox.com", notifier.GenericConfig.Endpoint)
	assert.Equal(t, "stackrox-ca-cert", notifier.GenericConfig.CACertPEM)
	assert.Equal(t, "Safest Password Generator", notifier.GenericConfig.Username)
	assert.Equal(t, "qwerty", notifier.GenericConfig.Password)
	assert.Len(t, notifier.GenericConfig.Headers, 2)
	assert.Equal(t, "Content-Length", notifier.GenericConfig.Headers[0].Key)
	assert.Equal(t, "120", notifier.GenericConfig.Headers[0].Value)
	assert.Equal(t, "Authorization", notifier.GenericConfig.Headers[1].Key)
	assert.Equal(t, "Basic", notifier.GenericConfig.Headers[1].Value)
	assert.Len(t, notifier.GenericConfig.ExtraFields, 1)
	assert.Equal(t, "extra", notifier.GenericConfig.ExtraFields[0].Key)
	assert.Equal(t, "field", notifier.GenericConfig.ExtraFields[0].Value)
	assert.True(t, notifier.GenericConfig.SkipTLSVerify)
	assert.True(t, notifier.GenericConfig.AuditLoggingEnabled)

	bytes, err := yaml.Marshal(&notifier)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}

func TestNotifierYAMLTransformation_Splunk(t *testing.T) {
	data := []byte(`name: test-name
splunk:
    token: stackrox-token
    endpoint: stackrox-endpoint
    skipTLSVerify: true
    auditLoggingEnabled: true
    hecTruncateLimit: 100
    sourceTypes:
        - key: audit
          sourceType: stackrox-audit
        - key: alert
          sourceType: stackrox-alert
`)
	notifier := Notifier{}

	err := yaml.Unmarshal(data, &notifier)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", notifier.Name)
	assert.NotNil(t, notifier.SplunkConfig)
	assert.Equal(t, "stackrox-token", notifier.SplunkConfig.HTTPToken)
	assert.Equal(t, "stackrox-endpoint", notifier.SplunkConfig.HTTPEndpoint)
	assert.True(t, notifier.SplunkConfig.AuditLoggingEnabled)
	assert.Equal(t, int64(100), notifier.SplunkConfig.Truncate)
	assert.Len(t, notifier.SplunkConfig.SourceTypes, 2)
	assert.Equal(t, "audit", notifier.SplunkConfig.SourceTypes[0].Key)
	assert.Equal(t, "stackrox-audit", notifier.SplunkConfig.SourceTypes[0].Value)
	assert.Equal(t, "alert", notifier.SplunkConfig.SourceTypes[1].Key)
	assert.Equal(t, "stackrox-alert", notifier.SplunkConfig.SourceTypes[1].Value)

	bytes, err := yaml.Marshal(&notifier)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}
