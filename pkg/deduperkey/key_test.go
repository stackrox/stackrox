package deduperkey

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stretchr/testify/suite"
)

const (
	stubID   = "2188759360372708523"
	stubHash = 2188759360372708523
)

var (
	stateWithAll = map[string]uint64{
		eventPkg.FormatKey("NetworkPolicy", fixtureconsts.NetworkPolicy1):   stubHash,
		eventPkg.FormatKey("Deployment", fixtureconsts.Deployment1):         stubHash,
		eventPkg.FormatKey("Pod", fixtureconsts.PodUID1):                    stubHash,
		eventPkg.FormatKey("Namespace", fixtureconsts.Namespace1):           stubHash,
		eventPkg.FormatKey("Secret", fixtureconsts.ServiceAccount1):         stubHash,
		eventPkg.FormatKey("Node", fixtureconsts.Node1):                     stubHash,
		eventPkg.FormatKey("ServiceAccount", fixtureconsts.ServiceAccount1): stubHash,
		eventPkg.FormatKey("Role", fixtureconsts.Role1):                     stubHash,
		eventPkg.FormatKey("Binding", fixtureconsts.RoleBinding1):           stubHash,
		eventPkg.FormatKey("NodeInventory", stubID):                         stubHash,
		eventPkg.FormatKey("ProcessIndicator", stubID):                      stubHash,
		eventPkg.FormatKey("ProviderMetadata", stubID):                      stubHash,
		eventPkg.FormatKey("OrchestratorMetadata", stubID):                  stubHash,
		eventPkg.FormatKey("ImageIntegration", stubID):                      stubHash,
		eventPkg.FormatKey("ComplianceOperatorResult", stubID):              stubHash,
		eventPkg.FormatKey("ComplianceOperatorProfile", stubID):             stubHash,
		eventPkg.FormatKey("ComplianceOperatorRule", stubID):                stubHash,
		eventPkg.FormatKey("ComplianceOperatorScanSettingBinding", stubID):  stubHash,
		eventPkg.FormatKey("ComplianceOperatorScan", stubID):                stubHash,
		eventPkg.FormatKey("AlertResults", stubID):                          stubHash,
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
				eventPkg.FormatKey("Deployment", fixtureconsts.Deployment1):             stubHash,
				fmt.Sprintf("%s_malformed_%s", "Deployment", fixtureconsts.Deployment1): stubHash,
			},
			expectedState: map[Key]uint64{
				withKey(&central.SensorEvent_Deployment{}, fixtureconsts.Deployment1): stubHash,
			},
		},
		"With invalid type entry": {
			inputState: map[string]uint64{
				eventPkg.FormatKey("Deployment", fixtureconsts.Deployment1):  stubHash,
				eventPkg.FormatKey("InvalidType", fixtureconsts.Deployment1): stubHash,
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
