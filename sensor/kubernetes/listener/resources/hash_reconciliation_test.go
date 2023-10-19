package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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
		"Unknown should throw error": {
			resType:       "Unknown",
			expectedMsg:   nil,
			expectedError: errors.New("Not implemented for resource type Unknown"),
		},
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
