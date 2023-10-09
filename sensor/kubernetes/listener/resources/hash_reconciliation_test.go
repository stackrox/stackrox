package resources

import (
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

func (s *HashReconciliationSuite) TestEmptyDeploymentProvider() {
	store := InitializeStore()
	rc := NewResourceStoreReconciler(store)

	// given the stores are empty, if you provide a single deduper entry, generate 1 delete event
	dstate := map[string]uint64{
		"Deployment:1234": 01234,
	}

	msgs := rc.ProcessHashes(dstate)

	s.Equal(1, len(msgs))
	s.Equal(msgs[0].GetEvent().Action, central.ResourceAction_REMOVE_RESOURCE)
	s.Equal(msgs[0].GetEvent().GetDeployment().GetId(), "1234")
}

func (s *HashReconciliationSuite) TestDeploymentProviderSingleDelete() {
	store := InitializeStore()
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("123"))
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("456"))
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("678"))
	rc := NewResourceStoreReconciler(store)

	// given the stores are empty, if you provide a single deduper entry, generate 1 delete event
	dstate := map[string]uint64{
		"Deployment:1234": 87654,
		"Deployment:456":  76543,
	}

	msgs := rc.ProcessHashes(dstate)

	s.Equal(1, len(msgs))
	s.Equal(msgs[0].GetEvent().Action, central.ResourceAction_REMOVE_RESOURCE)
	s.Equal(msgs[0].GetEvent().GetDeployment().GetId(), "1234")
}

func (s *HashReconciliationSuite) TestDeploymentProviderMultiDelete() {
	store := InitializeStore()
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("123"))
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("456"))
	store.deploymentStore.addOrUpdateDeployment(createWrapWithID("678"))
	rc := NewResourceStoreReconciler(store)

	// given the stores are empty, if you provide a single deduper entry, generate 1 delete event
	dstate := map[string]uint64{
		"Deployment:1234": 87654,
		"Deployment:456":  76543,
		"Deployment:0987": 98755,
	}

	msgs := rc.ProcessHashes(dstate)

	s.Equal(2, len(msgs))

	ids := make([]string, 0)
	for _, m := range msgs {
		s.Require().Equal(central.ResourceAction_REMOVE_RESOURCE, m.GetEvent().GetAction())
		ids = append(ids, m.GetEvent().GetDeployment().GetId())
	}

	s.ElementsMatch([]string{"0987", "1234"}, ids)
}

func createWrapWithID(id string) *deploymentWrap {
	d := createDeploymentWrap()
	d.Id = id
	return d
}
