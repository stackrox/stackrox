package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestAuthMachineToMachineConfig(t *testing.T) {
	data := []byte(`type: KUBE_SERVICE_ACCOUNT
tokenExpirationDuration: 1h
mappings:
    - key: sub
      value: system:serviceaccount:stackrox:config-controller
      role: Configuration Controller
issuer: https://kubernetes.default.svc
`)
	m2mConfig := &AuthMachineToMachineConfig{}

	err := yaml.Unmarshal(data, m2mConfig)
	assert.NoError(t, err)
	expectedType := AuthMachineToMachineConfigType(storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT)
	assert.Equal(t, expectedType, m2mConfig.Type)
	assert.Equal(t, "1h", m2mConfig.TokenExpirationDuration)
	assert.Equal(t, "https://kubernetes.default.svc", m2mConfig.Issuer)
	assert.Len(t, m2mConfig.Mappings, 1)
	if len(m2mConfig.Mappings) > 0 {
		mapping := m2mConfig.Mappings[0]
		assert.Equal(t, "sub", mapping.Key)
		assert.Equal(t, "system:serviceaccount:stackrox:config-controller", mapping.ValueExpression)
		assert.Equal(t, "Configuration Controller", mapping.Role)
	}

	bytes, err := yaml.Marshal(m2mConfig)
	assert.NoError(t, err)
	assert.Equal(t, string(data), string(bytes))
}

func TestAuthMachineToMachineConfigUnknownType(t *testing.T) {
	data := []byte(`type: true
tokenExpirationDuration: 1h
mappings:
    - key: sub
      value: system:serviceaccount:stackrox:config-controller
      role: Configuration Controller
issuer: https://kubernetes.default.svc
`)
	m2mConfig := &AuthMachineToMachineConfig{}

	err := yaml.Unmarshal(data, m2mConfig)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestAuthMachineToMachineConfigConfigurationType(t *testing.T) {
	authMachineToMachineObj := &AuthMachineToMachineConfig{}
	assert.Equal(t, AuthMachineToMachineConfiguration, authMachineToMachineObj.ConfigurationType())
}
