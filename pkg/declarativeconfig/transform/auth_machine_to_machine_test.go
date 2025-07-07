package transform

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformAuthMachineToMachine(t *testing.T) {
	transform := newAuthMachineToMachineConfigTransform()
	badObject := &declarativeconfig.AuthProvider{}
	messages, err := transform.Transform(badObject)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
	assert.Nil(t, messages)
}

func TestTransformAuthMachineToMachineConfig(t *testing.T) {
	transform := newAuthMachineToMachineConfigTransform()

	testConfigType := declarativeconfig.AuthMachineToMachineConfigType(
		storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
	)
	testTokenExpirationDuration := "1h"
	testIssuer := "https://kubernetes.default.svc"
	testMappingKey := "sub"
	testMappingValue := "system:serviceaccount:stackrox:config-controller"
	testRoleName := "Configuration Controller"
	testConfig := &declarativeconfig.AuthMachineToMachineConfig{
		ID:                      uuid.NewTestUUID(5).String(),
		Type:                    testConfigType,
		TokenExpirationDuration: testTokenExpirationDuration,
		Mappings: []declarativeconfig.MachineToMachineRoleMapping{
			{
				Key:             testMappingKey,
				ValueExpression: testMappingValue,
				Role:            testRoleName,
			},
		},
		Issuer: testIssuer,
	}

	output, err := transform.Transform(testConfig)
	assert.NoError(t, err)
	require.Len(t, output, 1)
	assert.Contains(t, output, authM2MConfigType)
	m2mMessages := output[authM2MConfigType]
	expectedOutputMsg := &storage.AuthMachineToMachineConfig{
		Id:                      declarativeconfig.NewDeclarativeM2MAuthConfigUUID(testIssuer).String(),
		Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		TokenExpirationDuration: testTokenExpirationDuration,
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             testMappingKey,
				ValueExpression: testMappingValue,
				Role:            testRoleName,
			},
		},
		Issuer: testIssuer,
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}
	assert.Len(t, m2mMessages, 1)
	if len(m2mMessages) > 0 {
		firstMessage := m2mMessages[0]
		actualOutputMsg, ok := firstMessage.(*storage.AuthMachineToMachineConfig)
		require.True(t, ok)
		protoassert.Equal(t, expectedOutputMsg, actualOutputMsg)
	}
}
