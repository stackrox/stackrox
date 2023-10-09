package resources

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
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
		s.T().Run(name, func(t *testing.T) {
			actual, err := resourceToMessage(c.resType, testResID)
			if c.expectedError != nil {
				require.Error(t, err)
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

func (s *HashReconciliationSuite) TestDeplProvider() {
	cases := map[string]struct {
		dstate     map[string]uint64
		deletedIDs []string
	}{
		"Deployment": {
			dstate: map[string]uint64{
				"Deployment:99": 87654,
				"Deployment:1":  76543,
			},
			deletedIDs: []string{"99"},
		},
		"Pod": {
			dstate: map[string]uint64{
				"Pod:99": 87654,
				"Pod:3":  76543,
			},
			deletedIDs: []string{"99"},
		},
	}

	for n, c := range cases {
		s.Run(n, func() {
			rc := NewResourceStoreReconciler(initStore())
			msgs := rc.ProcessHashes(c.dstate)

			s.Equal(len(c.deletedIDs), len(msgs))

			ids := make([]string, 0)
			for _, m := range msgs {
				s.Require().Equal(central.ResourceAction_REMOVE_RESOURCE, m.GetEvent().GetAction())
				idfn, err := resourceTypeToFn(reflect.TypeOf(m.GetEvent().GetResource()).String())
				s.Require().NoError(err)
				ids = append(ids, idfn(m.GetEvent()))
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
