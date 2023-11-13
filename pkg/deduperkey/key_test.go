package deduperkey

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/suite"
)

const (
	stubID   = "2188759360372708523"
	stubHash = 2188759360372708523
)

var (
	stateWithAll = map[string]uint64{
		fmt.Sprintf("%s:%s", "NetworkPolicy", fixtureconsts.NetworkPolicy1):   stubHash,
		fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):         stubHash,
		fmt.Sprintf("%s:%s", "Pod", fixtureconsts.PodUID1):                    stubHash,
		fmt.Sprintf("%s:%s", "Namespace", fixtureconsts.Namespace1):           stubHash,
		fmt.Sprintf("%s:%s", "Secret", fixtureconsts.ServiceAccount1):         stubHash,
		fmt.Sprintf("%s:%s", "Node", fixtureconsts.Node1):                     stubHash,
		fmt.Sprintf("%s:%s", "ServiceAccount", fixtureconsts.ServiceAccount1): stubHash,
		fmt.Sprintf("%s:%s", "Role", fixtureconsts.Role1):                     stubHash,
		fmt.Sprintf("%s:%s", "Binding", fixtureconsts.RoleBinding1):           stubHash,
		fmt.Sprintf("%s:%s", "NodeInventory", stubID):                         stubHash,
		fmt.Sprintf("%s:%s", "ProcessIndicator", stubID):                      stubHash,
		fmt.Sprintf("%s:%s", "ProviderMetadata", stubID):                      stubHash,
		fmt.Sprintf("%s:%s", "OrchestratorMetadata", stubID):                  stubHash,
		fmt.Sprintf("%s:%s", "ImageIntegration", stubID):                      stubHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorResult", stubID):              stubHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorProfile", stubID):             stubHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorRule", stubID):                stubHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorScanSettingBinding", stubID):  stubHash,
		fmt.Sprintf("%s:%s", "ComplianceOperatorScan", stubID):                stubHash,
		fmt.Sprintf("%s:%s", "AlertResults", stubID):                          stubHash,
	}
	expectedStateWithAll = map[Key]uint64{
		withKey(&central.SensorEvent_NetworkPolicy{}, fixtureconsts.NetworkPolicy1):   stubHash,
		withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1):         stubHash,
		withKey(&central.SensorEvent_Pod{}, fixtureconsts.PodUID1):                    stubHash,
		withKey(&central.SensorEvent_Namespace{}, fixtureconsts.Namespace1):           stubHash,
		withKey(&central.SensorEvent_Secret{}, fixtureconsts.ServiceAccount1):         stubHash,
		withKey(&central.SensorEvent_Node{}, fixtureconsts.Node1):                     stubHash,
		withKey(&central.SensorEvent_ServiceAccount{}, fixtureconsts.ServiceAccount1): stubHash,
		withKey(&central.SensorEvent_Role{}, fixtureconsts.Role1):                     stubHash,
		withKey(&central.SensorEvent_Binding{}, fixtureconsts.RoleBinding1):           stubHash,
		withKey(&central.SensorEvent_NodeInventory{}, stubID):                         stubHash,
		withKey(&central.SensorEvent_ProcessIndicator{}, stubID):                      stubHash,
		withKey(&central.SensorEvent_ProviderMetadata{}, stubID):                      stubHash,
		withKey(&central.SensorEvent_OrchestratorMetadata{}, stubID):                  stubHash,
		withKey(&central.SensorEvent_ImageIntegration{}, stubID):                      stubHash,
		withKey(&central.SensorEvent_ComplianceOperatorResult{}, stubID):              stubHash,
		withKey(&central.SensorEvent_ComplianceOperatorProfile{}, stubID):             stubHash,
		withKey(&central.SensorEvent_ComplianceOperatorRule{}, stubID):                stubHash,
		withKey(&central.SensorEvent_ComplianceOperatorScanSettingBinding{}, stubID):  stubHash,
		withKey(&central.SensorEvent_ComplianceOperatorScan{}, stubID):                stubHash,
		withKey(&central.SensorEvent_AlertResults{}, stubID):                          stubHash,
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
				fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):           stubHash,
				fmt.Sprintf("%s_malformed_%s", "Deployment", fixtureconsts.Deployment1): stubHash,
			},
			expectedState: map[Key]uint64{
				withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1): stubHash,
			},
		},
		"With invalid type entry": {
			inputState: map[string]uint64{
				fmt.Sprintf("%s:%s", "Deployment", fixtureconsts.Deployment1):  stubHash,
				fmt.Sprintf("%s:%s", "InvalidType", fixtureconsts.Deployment1): stubHash,
			},
			expectedState: map[Key]uint64{
				withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1): stubHash,
			},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			resultState := ParseDeduperState(tc.inputState)
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
