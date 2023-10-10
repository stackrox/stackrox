package resources

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
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
			resType:       "Pod",
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Pod{Pod: &storage.Pod{Id: testResID}}}},
			expectedError: nil,
		},
		"Deployment": {
			resType:       "Deployment",
			expectedMsg:   &central.MsgFromSensor_Event{Event: &central.SensorEvent{Id: testResID, Action: central.ResourceAction_REMOVE_RESOURCE, Resource: &central.SensorEvent_Deployment{Deployment: &storage.Deployment{Id: testResID}}}},
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
	case "*central.SensorEvent_Deployment":
		return func(event *central.SensorEvent) string {
			return event.GetDeployment().GetId()
		}, nil
	case "*central.SensorEvent_Pod":
		return func(event *central.SensorEvent) string {
			return event.GetPod().GetId()
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
	return s
}

func makeKey(id string, t reflect.Type) Key {
	return Key{id, t}
}

func (s *HashReconciliationSuite) TestProcessHashes() {
	cases := map[string]struct {
		dstate     map[Key]uint64
		deletedIDs []string
	}{
		"No Deployment": {
			dstate: map[Key]uint64{
				makeKey("1", TypeDeployment): 76543,
				makeKey("2", TypeDeployment): 76543,
			},
			deletedIDs: []string{},
		},
		"Single Deployment": {
			dstate: map[Key]uint64{
				makeKey("99", TypeDeployment): 87654,
				makeKey("1", TypeDeployment):  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Deployments": {
			dstate: map[Key]uint64{
				makeKey("99", TypeDeployment): 87654,
				makeKey("98", TypeDeployment): 88888,
				makeKey("97", TypeDeployment): 77777,
				makeKey("1", TypeDeployment):  76543,
			},
			deletedIDs: []string{"99", "98", "97"},
		},
		"No Pod": {
			dstate: map[Key]uint64{
				makeKey("3", TypePod): 76543,
				makeKey("4", TypePod): 76543,
			},
			deletedIDs: []string{},
		},
		"Single Pod": {
			dstate: map[Key]uint64{
				makeKey("99", TypePod): 87654,
				makeKey("3", TypePod):  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Multiple Pods": {
			dstate: map[Key]uint64{
				makeKey("99", TypePod):  87654,
				makeKey("100", TypePod): 87654,
				makeKey("101", TypePod): 87654,
				makeKey("3", TypePod):   76543,
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
				getIdFn, err := resourceTypeToFn(reflect.TypeOf(m.GetEvent().GetResource()).String())
				s.Require().NoError(err)
				ids = append(ids, getIdFn(m.GetEvent()))
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
