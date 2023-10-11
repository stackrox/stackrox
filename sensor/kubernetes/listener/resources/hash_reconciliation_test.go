package resources

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stretchr/testify/suite"
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
		"ServiceAccount": {
			resType:       deduper.TypeServiceAccount.String(),
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_ServiceAccount{ServiceAccount: &storage.ServiceAccount{Id: testResID}}}},
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
	return s
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
