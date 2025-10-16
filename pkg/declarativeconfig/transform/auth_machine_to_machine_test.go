package transform

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
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

	const (
		testConfigType = declarativeconfig.AuthMachineToMachineConfigType(
			storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		)
		testTokenExpirationDuration = "1h"
		testIssuer                  = "https://kubernetes.default.svc"
		testMappingKey              = "sub"
		testMappingValue            = "system:serviceaccount:stackrox:config-controller"
		testRoleName                = "Configuration Controller"
	)

	testConfig := &declarativeconfig.AuthMachineToMachineConfig{
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

	am := &storage.AuthMachineToMachineConfig_Mapping{}
	am.SetKey(testMappingKey)
	am.SetValueExpression(testMappingValue)
	am.SetRole(testRoleName)
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	expectedOutputMsg := &storage.AuthMachineToMachineConfig{}
	expectedOutputMsg.SetId(declarativeconfig.NewDeclarativeM2MAuthConfigUUID(testIssuer).String())
	expectedOutputMsg.SetType(storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT)
	expectedOutputMsg.SetTokenExpirationDuration(testTokenExpirationDuration)
	expectedOutputMsg.SetMappings([]*storage.AuthMachineToMachineConfig_Mapping{
		am,
	})
	expectedOutputMsg.SetIssuer(testIssuer)
	expectedOutputMsg.SetTraits(traits)

	castedTransformOutput := make([]*storage.AuthMachineToMachineConfig, 0, len(m2mMessages))
	for _, m := range m2mMessages {
		casted, ok := m.(*storage.AuthMachineToMachineConfig)
		if ok {
			castedTransformOutput = append(castedTransformOutput, casted)
		}
	}

	protoassert.SlicesEqual(t, []*storage.AuthMachineToMachineConfig{expectedOutputMsg}, castedTransformOutput)
}

func TestUniversalTransformAuthMachineToMachineConfig(t *testing.T) {
	transform := New()

	const (
		testConfigType = declarativeconfig.AuthMachineToMachineConfigType(
			storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		)
		testTokenExpirationDuration = "1h"
		testIssuer                  = "https://kubernetes.default.svc"
		testMappingKey              = "sub"
		testMappingValue            = "system:serviceaccount:stackrox:config-controller"
		testRoleName                = "Configuration Controller"
	)

	testConfig := &declarativeconfig.AuthMachineToMachineConfig{
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

	am := &storage.AuthMachineToMachineConfig_Mapping{}
	am.SetKey(testMappingKey)
	am.SetValueExpression(testMappingValue)
	am.SetRole(testRoleName)
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	expectedOutputMsg := &storage.AuthMachineToMachineConfig{}
	expectedOutputMsg.SetId(declarativeconfig.NewDeclarativeM2MAuthConfigUUID(testIssuer).String())
	expectedOutputMsg.SetType(storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT)
	expectedOutputMsg.SetTokenExpirationDuration(testTokenExpirationDuration)
	expectedOutputMsg.SetMappings([]*storage.AuthMachineToMachineConfig_Mapping{
		am,
	})
	expectedOutputMsg.SetIssuer(testIssuer)
	expectedOutputMsg.SetTraits(traits)

	castedTransformOutput := make([]*storage.AuthMachineToMachineConfig, 0, len(m2mMessages))
	for _, m := range m2mMessages {
		casted, ok := m.(*storage.AuthMachineToMachineConfig)
		if ok {
			castedTransformOutput = append(castedTransformOutput, casted)
		}
	}

	protoassert.SlicesEqual(t, []*storage.AuthMachineToMachineConfig{expectedOutputMsg}, castedTransformOutput)
}
