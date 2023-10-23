package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestHashReconciliation(t *testing.T) {
	suite.Run(t, new(HashReconciliationSuite))
}

type HashReconciliationSuite struct {
	suite.Suite
}

var testResID = uuid.NewDummy().String()

func (s *HashReconciliationSuite) TestResourceToMessage() {
	cases := map[string]struct {
		resType       string
		expectedMsg   *central.MsgFromSensor_Event
		expectedError error
	}{
		"Pod": {
			resType:       deduper.TypePod.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Pod{Pod: &storage.Pod{Id: testResID}}}},
			expectedError: nil,
		},
		"Deployment": {
			resType:       deduper.TypeDeployment.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Deployment{Deployment: &storage.Deployment{Id: testResID}}}},
			expectedError: nil,
		},
		"Node": {
			resType:       deduper.TypeNode.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Node{Node: &storage.Node{Id: testResID}}}},
			expectedError: nil,
		},
		"ServiceAccount": {
			resType:       deduper.TypeServiceAccount.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ServiceAccount{ServiceAccount: &storage.ServiceAccount{Id: testResID}}}},
			expectedError: nil,
		},
		"Secret": {
			resType:       deduper.TypeSecret.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Secret{Secret: &storage.Secret{Id: testResID}}}},
			expectedError: nil,
		},
		"NetworkPolicy": {
			resType:       deduper.TypeNetworkPolicy.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_NetworkPolicy{NetworkPolicy: &storage.NetworkPolicy{Id: testResID}}}},
			expectedError: nil,
		},
		"Role": {
			resType:       deduper.TypeRole.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Role{Role: &storage.K8SRole{Id: testResID}}}},
			expectedError: nil,
		},
		"Binding": {
			resType:       deduper.TypeBinding.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Binding{Binding: &storage.K8SRoleBinding{Id: testResID}}}},
			expectedError: nil,
		},
		"Namespace": {
			resType:       deduper.TypeNamespace.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Namespace{Namespace: &storage.NamespaceMetadata{Id: testResID}}}},
			expectedError: nil,
		},
		"ComplianceOperatorProfile": {
			resType:       deduper.TypeComplianceOperatorProfile.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorProfile{ComplianceOperatorProfile: &storage.ComplianceOperatorProfile{Id: testResID}}}},
			expectedError: nil,
		},
		"ComplianceOperatorRule": {
			resType:       deduper.TypeComplianceOperatorRule.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorRule{ComplianceOperatorRule: &storage.ComplianceOperatorRule{Id: testResID}}}},
			expectedError: nil,
		},
		"ComplianceOperatorScan": {
			resType:       deduper.TypeComplianceOperatorScan.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorScan{ComplianceOperatorScan: &storage.ComplianceOperatorScan{Id: testResID}}}},
			expectedError: nil,
		},
		"ComplianceOperatorScanSettingBinding": {
			resType:       deduper.TypeComplianceOperatorScanSettingBinding.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{Id: testResID}}}},
			expectedError: nil,
		},
		"Unknown should throw error": {
			resType:       "Unknown",
			expectedMsg:   nil,
			expectedError: errors.New("Not implemented for resource type Unknown"),
		},
	}
	if features.ComplianceEnhancements.Enabled() {
		cases["ComplianceOperatorResults"] = struct {
			resType       string
			expectedMsg   *central.MsgFromSensor_Event
			expectedError error
		}{
			resType:       deduper.TypeComplianceOperatorResult.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorResultV2{ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{Id: testResID}}}},
			expectedError: nil,
		}
	} else {
		cases["ComplianceOperatorResults"] = struct {
			resType       string
			expectedMsg   *central.MsgFromSensor_Event
			expectedError error
		}{
			resType:       deduper.TypeComplianceOperatorResult.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ComplianceOperatorResult{ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{Id: testResID}}}},
			expectedError: nil,
		}
	}

	for name, c := range cases {
		s.Run(name, func() {
			actual, err := resourceToMessage(c.resType, testResID)
			if c.expectedError != nil {
				s.Require().Error(err)
				return
			}
			s.Equal(c.expectedMsg, actual.Msg)
			s.NoError(err)
		})
	}
}

func resourceTypeToFn(resType string) (func(*central.SensorEvent) string, error) {
	switch resType {
	case deduper.TypeDeployment.String():
		return func(event *central.SensorEvent) string {
			return event.GetDeployment().GetId()
		}, nil
	case deduper.TypePod.String():
		return func(event *central.SensorEvent) string {
			return event.GetPod().GetId()
		}, nil
	case deduper.TypeServiceAccount.String():
		return func(event *central.SensorEvent) string {
			return event.GetServiceAccount().GetId()
		}, nil
	case deduper.TypeSecret.String():
		return func(event *central.SensorEvent) string {
			return event.GetSecret().GetId()
		}, nil
	case deduper.TypeNode.String():
		return func(event *central.SensorEvent) string {
			return event.GetNode().GetId()
		}, nil
	case deduper.TypeNetworkPolicy.String():
		return func(event *central.SensorEvent) string {
			return event.GetNetworkPolicy().GetId()
		}, nil
	case deduper.TypeRole.String():
		return func(event *central.SensorEvent) string {
			return event.GetRole().GetId()
		}, nil
	case deduper.TypeBinding.String():
		return func(event *central.SensorEvent) string {
			return event.GetBinding().GetId()
		}, nil
	case deduper.TypeNamespace.String():
		return func(event *central.SensorEvent) string {
			return event.GetNamespace().GetId()
		}, nil
	case deduper.TypeComplianceOperatorProfile.String():
		return func(event *central.SensorEvent) string {
			return event.GetComplianceOperatorProfile().GetId()
		}, nil
	case deduper.TypeComplianceOperatorResult.String():
		if features.ComplianceEnhancements.Enabled() {
			return func(event *central.SensorEvent) string {
				return event.GetComplianceOperatorResultV2().GetId()
			}, nil
		}
		return func(event *central.SensorEvent) string {
			return event.GetComplianceOperatorResult().GetId()
		}, nil
	case deduper.TypeComplianceOperatorRule.String():
		return func(event *central.SensorEvent) string {
			return event.GetComplianceOperatorRule().GetId()
		}, nil
	case deduper.TypeComplianceOperatorScan.String():
		return func(event *central.SensorEvent) string {
			return event.GetComplianceOperatorScan().GetId()
		}, nil
	case deduper.TypeComplianceOperatorScanSettingBinding.String():
		return func(event *central.SensorEvent) string {
			return event.GetComplianceOperatorScanSettingBinding().GetId()
		}, nil
	default:
		return nil, errors.Errorf("not implemented for resource type %v", resType)
	}

}

func initStore() *InMemoryStoreProvider {
	s := InitializeStore()
	s.deploymentStore.addOrUpdateDeployment(createWrapWithID("1"))
	s.deploymentStore.addOrUpdateDeployment(createWrapWithID("2"))
	s.podStore.addOrUpdatePod(&storage.Pod{Id: "3"})
	s.podStore.addOrUpdatePod(&storage.Pod{Id: "4"})
	s.nodeStore.addOrUpdateNode(makeNode("42"))
	s.nodeStore.addOrUpdateNode(makeNode("43"))
	s.networkPolicyStore.Upsert(&storage.NetworkPolicy{Id: "1"})
	s.networkPolicyStore.Upsert(&storage.NetworkPolicy{Id: "2"})
	s.serviceAccountStore.Add(&storage.ServiceAccount{
		Id:               "5",
		Name:             "Acc1",
		Namespace:        "Test",
		ImagePullSecrets: []string{},
	})
	s.serviceAccountStore.Add(&storage.ServiceAccount{
		Id:               "6",
		Name:             "Acc2",
		Namespace:        "Test",
		ImagePullSecrets: []string{},
	})
	s.registryStore.AddSecretID("5000")
	s.registryStore.AddSecretID("5001")

	s.rbacStore.UpsertRole(&rbacV1.Role{
		ObjectMeta: metav1.ObjectMeta{UID: "6001", Namespace: "a", Name: "6001"},
	})
	s.rbacStore.UpsertClusterRole(&rbacV1.ClusterRole{ObjectMeta: metav1.ObjectMeta{UID: "6002", Name: "6002"}})
	s.rbacStore.UpsertBinding(&rbacV1.RoleBinding{ObjectMeta: metav1.ObjectMeta{UID: "6003"}})
	s.rbacStore.UpsertClusterBinding(&rbacV1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{UID: "6004"}})

	s.nsStore.addNamespace(&storage.NamespaceMetadata{Id: "1", Name: "a"})
	s.nsStore.addNamespace(&storage.NamespaceMetadata{Id: "2", Name: "b"})

	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorProfile.String(), "1")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorProfile.String(), "2")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorResult.String(), "1")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorResult.String(), "2")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorRule.String(), "1")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorRule.String(), "2")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScan.String(), "1")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScan.String(), "2")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScanSettingBinding.String(), "1")
	s.reconciliationStore.Upsert(deduper.TypeComplianceOperatorScanSettingBinding.String(), "2")
	return s
}

func makeNode(id types.UID) *nodeWrap {
	return &nodeWrap{
		Node: &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("Node-%s", id),
				UID:  id,
			},
		},
	}
}

func makeKey(id string, t reflect.Type) deduper.Key {
	return deduper.Key{ID: id, ResourceType: t}
}

func (s *HashReconciliationSuite) TestProcessHashes() {
	cases := map[string]struct {
		dstate     map[deduper.Key]uint64
		deletedIDs []string
	}{
		"No Deployment": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeDeployment): 76543,
				makeKey("2", deduper.TypeDeployment): 76543,
			},
			deletedIDs: []string{},
		},
		"Single Deployment": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeDeployment): 87654,
				makeKey("1", deduper.TypeDeployment):  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Deployments": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeDeployment): 87654,
				makeKey("98", deduper.TypeDeployment): 88888,
				makeKey("97", deduper.TypeDeployment): 77777,
				makeKey("1", deduper.TypeDeployment):  76543,
			},
			deletedIDs: []string{"99", "98", "97"},
		},
		"No Pod": {
			dstate: map[deduper.Key]uint64{
				makeKey("3", deduper.TypePod): 76543,
				makeKey("4", deduper.TypePod): 76543,
			},
			deletedIDs: []string{},
		},
		"Single Pod": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypePod): 87654,
				makeKey("3", deduper.TypePod):  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Pods": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypePod):  87654,
				makeKey("100", deduper.TypePod): 87654,
				makeKey("101", deduper.TypePod): 87654,
				makeKey("3", deduper.TypePod):   76543,
			},
			deletedIDs: []string{"99", "100", "101"},
		},
		"No ServiceAccount": {
			dstate: map[deduper.Key]uint64{
				makeKey("5", deduper.TypeServiceAccount): 76543,
				makeKey("6", deduper.TypeServiceAccount): 65432,
			},
			deletedIDs: []string{},
		},
		"Single ServiceAccount": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeServiceAccount): 87654,
				makeKey("5", deduper.TypeServiceAccount):  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple ServiceAccounts": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeServiceAccount):  87654,
				makeKey("100", deduper.TypeServiceAccount): 87654,
				makeKey("101", deduper.TypeServiceAccount): 87654,
				makeKey("5", deduper.TypeServiceAccount):   76543,
			},
			deletedIDs: []string{"99", "100", "101"},
		},
		"No Secret": {
			dstate: map[deduper.Key]uint64{
				makeKey("5000", deduper.TypeSecret): 76543,
				makeKey("5001", deduper.TypeSecret): 65432,
			},
			deletedIDs: []string{},
		},
		"Single Secret": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeSecret):   87654,
				makeKey("5000", deduper.TypeSecret): 76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Secrets": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeSecret):   87654,
				makeKey("100", deduper.TypeSecret):  87654,
				makeKey("101", deduper.TypeSecret):  87654,
				makeKey("5000", deduper.TypeSecret): 76543,
			},
			deletedIDs: []string{"99", "100", "101"},
		},
		"No Node": {
			dstate: map[deduper.Key]uint64{
				makeKey("42", deduper.TypeNode): 87654,
				makeKey("43", deduper.TypeNode): 76543,
			},
			deletedIDs: []string{},
		},
		"Single Node": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNode): 87654,
				makeKey("42", deduper.TypeNode): 76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Nodes": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNode): 87654,
				makeKey("98", deduper.TypeNode): 33333,
				makeKey("97", deduper.TypeNode): 76654,
				makeKey("42", deduper.TypeNode): 76543,
			},
			deletedIDs: []string{"99", "98", "97"},
		},
		"No Network Policy": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeNetworkPolicy): 12345,
				makeKey("2", deduper.TypeNetworkPolicy): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Network Policy": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNetworkPolicy): 34567,
				makeKey("1", deduper.TypeNetworkPolicy):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Network Policies": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNetworkPolicy): 34567,
				makeKey("98", deduper.TypeNetworkPolicy): 34567,
				makeKey("97", deduper.TypeNetworkPolicy): 34567,
				makeKey("1", deduper.TypeNetworkPolicy):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No RBACs": {
			dstate: map[deduper.Key]uint64{
				makeKey("6001", deduper.TypeRole):    76543,
				makeKey("6003", deduper.TypeBinding): 87654,
			},
			deletedIDs: []string{},
		},
		"One role, one binding": {
			dstate: map[deduper.Key]uint64{
				makeKey("6002", deduper.TypeRole):    76543,
				makeKey("6004", deduper.TypeBinding): 87654,
				makeKey("99", deduper.TypeRole):      76543,
				makeKey("98", deduper.TypeBinding):   87654,
			},
			deletedIDs: []string{"99", "98"},
		},
		"No Namespace": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeNamespace): 12345,
				makeKey("2", deduper.TypeNamespace): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Namespace": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNamespace): 34567,
				makeKey("1", deduper.TypeNamespace):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Namespaces": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeNamespace): 34567,
				makeKey("98", deduper.TypeNamespace): 34567,
				makeKey("97", deduper.TypeNamespace): 34567,
				makeKey("1", deduper.TypeNamespace):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No Compliance Operator Profile": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeComplianceOperatorProfile): 12345,
				makeKey("2", deduper.TypeComplianceOperatorProfile): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Compliance Operator Profile": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorProfile): 34567,
				makeKey("1", deduper.TypeComplianceOperatorProfile):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Compliance Operator Profiles": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorProfile): 34567,
				makeKey("98", deduper.TypeComplianceOperatorProfile): 34567,
				makeKey("97", deduper.TypeComplianceOperatorProfile): 34567,
				makeKey("1", deduper.TypeComplianceOperatorProfile):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No Compliance Operator Result": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeComplianceOperatorResult): 12345,
				makeKey("2", deduper.TypeComplianceOperatorResult): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Compliance Operator Result": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorResult): 34567,
				makeKey("1", deduper.TypeComplianceOperatorResult):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Compliance Operator Results": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorResult): 34567,
				makeKey("98", deduper.TypeComplianceOperatorResult): 34567,
				makeKey("97", deduper.TypeComplianceOperatorResult): 34567,
				makeKey("1", deduper.TypeComplianceOperatorResult):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No Compliance Operator Rule": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeComplianceOperatorRule): 12345,
				makeKey("2", deduper.TypeComplianceOperatorRule): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Compliance Operator Rule": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorRule): 34567,
				makeKey("1", deduper.TypeComplianceOperatorRule):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Compliance Operator Rules": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorRule): 34567,
				makeKey("98", deduper.TypeComplianceOperatorRule): 34567,
				makeKey("97", deduper.TypeComplianceOperatorRule): 34567,
				makeKey("1", deduper.TypeComplianceOperatorRule):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No Compliance Operator Scan": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeComplianceOperatorScan): 12345,
				makeKey("2", deduper.TypeComplianceOperatorScan): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Compliance Operator Scan": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorScan): 34567,
				makeKey("1", deduper.TypeComplianceOperatorScan):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Compliance Operator Scans": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorScan): 34567,
				makeKey("98", deduper.TypeComplianceOperatorScan): 34567,
				makeKey("97", deduper.TypeComplianceOperatorScan): 34567,
				makeKey("1", deduper.TypeComplianceOperatorScan):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
		"No Compliance Operator Scan Setting Binding": {
			dstate: map[deduper.Key]uint64{
				makeKey("1", deduper.TypeComplianceOperatorScanSettingBinding): 12345,
				makeKey("2", deduper.TypeComplianceOperatorScanSettingBinding): 34567,
			},
			deletedIDs: []string{},
		},
		"Single Compliance Operator Scan Setting Binding": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorScanSettingBinding): 34567,
				makeKey("1", deduper.TypeComplianceOperatorScanSettingBinding):  12345,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Compliance Operator Scan Setting Bindings": {
			dstate: map[deduper.Key]uint64{
				makeKey("99", deduper.TypeComplianceOperatorScanSettingBinding): 34567,
				makeKey("98", deduper.TypeComplianceOperatorScanSettingBinding): 34567,
				makeKey("97", deduper.TypeComplianceOperatorScanSettingBinding): 34567,
				makeKey("1", deduper.TypeComplianceOperatorScanSettingBinding):  12345,
			},
			deletedIDs: []string{"97", "98", "99"},
		},
	}

	for n, c := range cases {
		s.Run(n, func() {
			rc := NewResourceStoreReconciler(initStore())
			msgs := rc.ProcessHashes(c.dstate)

			s.Len(msgs, len(c.deletedIDs))

			ids := make([]string, 0)
			for _, m := range msgs {
				s.Require().Equal(central.ResourceAction_REMOVE_RESOURCE, m.GetEvent().GetAction())
				getIDFn, err := resourceTypeToFn(reflect.TypeOf(m.GetEvent().GetResource()).String())
				s.Require().NoError(err)
				ids = append(ids, getIDFn(m.GetEvent()))
			}
			s.ElementsMatch(c.deletedIDs, ids)
		})
	}
}

func createWrapWithID(id string) *deploymentWrap {
	d := createDeploymentWrap()
	d.Id = id
	return d
}
