package policyrefresh

import (
	"testing"

	"github.com/golang/mock/gomock"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	detectionMocks "github.com/stackrox/rox/central/detection/mocks"
	policyDataStoreMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockPolicies  *policyDataStoreMocks.MockDataStore
	mockManager   *lifecycleMocks.MockManager
	mockPolicySet *detectionMocks.MockPolicySet

	tested pipeline.Fragment
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockPolicies = policyDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockManager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.mockPolicySet = detectionMocks.NewMockPolicySet(suite.mockCtrl)

	suite.tested = &pipelineImpl{
		policies:                suite.mockPolicies,
		deployAndRuntimeManager: suite.mockManager,
		buildTimePolicies:       suite.mockPolicySet,
		throttler:               &dontThrottle{},
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestUpdatesAllMatchingPolicies() {
	policies := []*storage.Policy{
		{
			Id: "p1",
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_BUILD,
				storage.LifecycleStage_DEPLOY,
				storage.LifecycleStage_RUNTIME,
			},
			Fields: &storage.PolicyFields{
				PermissionPolicy: &storage.PermissionPolicy{
					PermissionLevel: storage.PermissionLevel_NONE,
				},
			},
		},
	}
	suite.mockPolicies.EXPECT().GetPolicies(gomock.Any()).Return(policies, nil)

	// Expect manager to be updated.
	suite.mockManager.EXPECT().RecompilePolicy(policies[0]).Return(nil)
	suite.mockPolicySet.EXPECT().Recompile(policies[0].GetId()).Return(nil)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ServiceAccount{},
			},
		},
	}
	err := suite.tested.Run("", msg, nil)
	suite.NoError(err, "expected the error")
}

func (suite *PipelineTestSuite) TestFiltersMatchingPolicies() {
	policies := []*storage.Policy{
		{
			Id: "p1",
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_BUILD,
				storage.LifecycleStage_DEPLOY,
				storage.LifecycleStage_RUNTIME,
			},
		},
	}
	suite.mockPolicies.EXPECT().GetPolicies(gomock.Any()).Return(policies, nil)

	// Expect manager and policy to not be updated since the RBAC field is not present in any policy.

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ServiceAccount{},
			},
		},
	}
	err := suite.tested.Run("", msg, nil)
	suite.NoError(err, "expected the error")
}

func (suite *PipelineTestSuite) TestMatches() {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{},
	}
	suite.Equal(false, suite.tested.Match(msg), "expected message not to match")

	msg = &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Deployment{},
			},
		},
	}
	suite.Equal(false, suite.tested.Match(msg), "expected message not to match")

	msg = &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ServiceAccount{
					ServiceAccount: &storage.ServiceAccount{},
				},
			},
		},
	}
	suite.Equal(true, suite.tested.Match(msg), "expected message to match")

	msg = &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Role{
					Role: &storage.K8SRole{},
				},
			},
		},
	}
	suite.Equal(true, suite.tested.Match(msg), "expected message to match")

	msg = &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Binding{
					Binding: &storage.K8SRoleBinding{},
				},
			},
		},
	}
	suite.Equal(true, suite.tested.Match(msg), "expected message to match")
}

// skip throttling.
type dontThrottle struct{}

func (dt *dontThrottle) Run(f func()) {
	f()
}
