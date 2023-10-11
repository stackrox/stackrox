package deduper

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/suite"
)

const (
	randomID   = "2188759360372708523"
	randomHash = 2188759360372708523
)

var (
	stateWithAll = map[string]uint64{
		fmt.Sprintf("%s:%s", "NetworkPolicy", fixtureconsts.NetworkPolicy1):    randomHash,
		fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):          randomHash,
		fmt.Sprintf("%s:%s", "Pod", fixtureconsts.PodUID1):                     randomHash,
		fmt.Sprintf("%s:%s", "Namespace", fixtureconsts.Namespace1):            randomHash,
		fmt.Sprintf("%s:%s", "Secret", fixtureconsts.ServiceAccount1):          randomHash,
		fmt.Sprintf("%s:%s", "Node", fixtureconsts.Node1):                      randomHash,
		fmt.Sprintf("%s:%s", "ServiceAccount", fixtureconsts.ServiceAccount1):  randomHash,
		fmt.Sprintf("%s:%s", "Role", fixtureconsts.Role1):                      randomHash,
		fmt.Sprintf("%s:%s", "Binding", fixtureconsts.RoleBinding1):            randomHash,
		fmt.Sprintf("%s:%s", "NodeInventory", randomID):                        randomHash,
		fmt.Sprintf("%s:%s", "ProcessIndicator", randomID):                     randomHash,
		fmt.Sprintf("%s:%s", "ProviderMetadata", randomID):                     randomHash,
		fmt.Sprintf("%s:%s", "OrchestratorMetadata", randomID):                 randomHash,
		fmt.Sprintf("%s:%s", "ImageIntegration", randomID):                     randomHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorResult", randomID):             randomHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorProfile", randomID):            randomHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorRule", randomID):               randomHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorScanSettingBinding", randomID): randomHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorScan", randomID):               randomHash,
		fmt.Sprintf("%s:%s", "AlertResults", randomID):                         randomHash,
	}
	expectedStateWithAll = map[Key]uint64{
		withKey(&central.SensorEvent_NetworkPolicy{}, fixtureconsts.NetworkPolicy1):    randomHash,
		withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1):          randomHash,
		withKey(&central.SensorEvent_Pod{}, fixtureconsts.PodUID1):                     randomHash,
		withKey(&central.SensorEvent_Namespace{}, fixtureconsts.Namespace1):            randomHash,
		withKey(&central.SensorEvent_Secret{}, fixtureconsts.ServiceAccount1):          randomHash,
		withKey(&central.SensorEvent_Node{}, fixtureconsts.Node1):                      randomHash,
		withKey(&central.SensorEvent_ServiceAccount{}, fixtureconsts.ServiceAccount1):  randomHash,
		withKey(&central.SensorEvent_Role{}, fixtureconsts.Role1):                      randomHash,
		withKey(&central.SensorEvent_Binding{}, fixtureconsts.RoleBinding1):            randomHash,
		withKey(&central.SensorEvent_NodeInventory{}, randomID):                        randomHash,
		withKey(&central.SensorEvent_ProcessIndicator{}, randomID):                     randomHash,
		withKey(&central.SensorEvent_ProviderMetadata{}, randomID):                     randomHash,
		withKey(&central.SensorEvent_OrchestratorMetadata{}, randomID):                 randomHash,
		withKey(&central.SensorEvent_ImageIntegration{}, randomID):                     randomHash,
		withKey(&central.SensorEvent_ComplianceOperatorResult{}, randomID):             randomHash,
		withKey(&central.SensorEvent_ComplianceOperatorProfile{}, randomID):            randomHash,
		withKey(&central.SensorEvent_ComplianceOperatorRule{}, randomID):               randomHash,
		withKey(&central.SensorEvent_ComplianceOperatorScanSettingBinding{}, randomID): randomHash,
		withKey(&central.SensorEvent_ComplianceOperatorScan{}, randomID):               randomHash,
		withKey(&central.SensorEvent_AlertResults{}, randomID):                         randomHash,
	}
)

type deduperKeySuite struct {
	suite.Suite
}

func Test_DeduperKeySuite(t *testing.T) {
	suite.Run(t, new(deduperKeySuite))

}

func (s *deduperKeySuite) Test_CopyState() {
	testCases := map[string]struct {
		inputState    map[string]uint64
		expectedState map[Key]uint64
	}{
		"All event types": {
			inputState:    stateWithAll,
			expectedState: expectedStateWithAll,
		},
		"Nil input": {
			inputState:    nil,
			expectedState: map[Key]uint64{},
		},
		"With malformed entry": {
			inputState: map[string]uint64{
				fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):           randomHash,
				fmt.Sprintf("%s_malformed_%s", "Deployment", fixtureconsts.Deployment1): randomHash,
			},
			expectedState: map[Key]uint64{
				withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1): randomHash,
			},
		},
		"With invalid type entry": {
			inputState: map[string]uint64{
				fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):  randomHash,
				fmt.Sprintf("%s:%s", "InvalidType", fixtureconsts.Deployment1): randomHash,
			},
			expectedState: map[Key]uint64{
				withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1): randomHash,
			},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			resultState := CopyDeduperState(tc.inputState)
			s.Assert().Equal(tc.expectedState, resultState)
		})
	}
}

func withKey(resource any, id string) Key {
	return Key{
		ID:           id,
		ResourceType: reflect.TypeOf(resource),
	}
}
