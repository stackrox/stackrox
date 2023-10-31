package deploymentevents

import (
	"context"
	"testing"

	aggregatorMocks "github.com/stackrox/rox/central/activecomponent/updater/aggregator/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	graphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	clusters          *clusterMocks.MockDataStore
	deployments       *deploymentMocks.MockDataStore
	networkBaselines  *networkBaselineMocks.MockManager
	manager           *lifecycleMocks.MockManager
	graphEvaluator    *graphMocks.MockEvaluator
	reprocessor       *reprocessorMocks.MockLoop
	pipeline          *pipelineImpl
	processAggregator *aggregatorMocks.MockProcessAggregator

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.networkBaselines = networkBaselineMocks.NewMockManager(suite.mockCtrl)
	suite.manager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.graphEvaluator = graphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.reprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)
	suite.processAggregator = aggregatorMocks.NewMockProcessAggregator(suite.mockCtrl)
	suite.pipeline =
		NewPipeline(
			suite.clusters,
			suite.deployments,
			suite.manager,
			suite.graphEvaluator,
			suite.reprocessor,
			suite.networkBaselines,
			suite.processAggregator).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestDeploymentSyncResources() {
	deployment := fixtures.GetDeployment()

	suite.clusters.EXPECT().GetClusterName(context.Background(), deployment.GetClusterId())
	suite.deployments.EXPECT().GetDeployment(context.Background(), deployment.GetId())
	suite.deployments.EXPECT().UpsertDeployment(context.Background(), gomock.Any())
	suite.processAggregator.EXPECT().RefreshDeployment(gomock.Any()).AnyTimes()
	suite.networkBaselines.EXPECT().ProcessDeploymentCreate(deployment.GetId(), deployment.GetName(), deployment.GetClusterId(), gomock.Any())
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())

	err := suite.pipeline.Run(context.Background(), deployment.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     deployment.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Deployment{
					Deployment: deployment,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestDeploymentRemovePipeline() {
	deployment := fixtures.GetDeployment()

	suite.deployments.EXPECT().RemoveDeployment(context.Background(), deployment.GetClusterId(), deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())
	suite.networkBaselines.EXPECT().ProcessDeploymentDelete(gomock.Any()).Return(nil)

	err := suite.pipeline.Run(context.Background(), deployment.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     deployment.GetId(),
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Deployment{
					Deployment: deployment,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestSensorReconcileDeploymentRemove() {
	deployment := fixtures.GetDeployment()

	suite.deployments.EXPECT().RemoveDeployment(context.Background(), deployment.GetClusterId(), deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())
	suite.networkBaselines.EXPECT().ProcessDeploymentDelete(gomock.Any()).Return(nil)

	err := suite.pipeline.Run(context.Background(), deployment.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     deployment.GetId(),
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{Id: deployment.GetId()},
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestCreateNetworkBaseline() {
	deployment := fixtures.GetDeployment()

	suite.clusters.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("cluster-name", true, nil)
	suite.deployments.EXPECT().UpsertDeployment(gomock.Any(), gomock.Any()).Return(nil)
	suite.networkBaselines.EXPECT().ProcessDeploymentCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(gomock.Any()).Return()
	suite.graphEvaluator.EXPECT().IncrementEpoch(gomock.Any()).Return()
	suite.processAggregator.EXPECT().RefreshDeployment(gomock.Any()).AnyTimes()

	err := suite.pipeline.Run(
		context.Background(),
		deployment.GetClusterId(),
		&central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Id:     deployment.GetId(),
					Action: central.ResourceAction_CREATE_RESOURCE,
					Resource: &central.SensorEvent_Deployment{
						Deployment: deployment,
					},
				},
			},
		},
		nil)
	suite.Nil(err)
}

func (suite *PipelineTestSuite) TestAlertRemovalOnReconciliation() {
	deployment := fixtures.GetDeployment()

	suite.deployments.EXPECT().RemoveDeployment(context.Background(), deployment.GetClusterId(), deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())
	suite.manager.EXPECT().DeploymentRemoved(deployment.GetId())
	suite.networkBaselines.EXPECT().ProcessDeploymentDelete(deployment.GetId()).Return(nil)

	suite.NoError(suite.pipeline.runRemovePipeline(context.Background(), deployment.GetId(), deployment.GetClusterId(), true))
}

func (suite *PipelineTestSuite) TestValidateImages() {
	events := fakeDeploymentEvents()

	// Call function.
	tested := &validateInputImpl{}

	// Pull one more time to get nil
	suite.NoError(tested.do(events[0].GetDeployment()), "valid input should not throw an error.")

	// Pull one more time to get nil
	suite.Error(tested.do(nil), "event without deployment should fail")

	// Pull one more time to get nil
	events[0] = nil
	suite.Error(tested.do(events[0].GetDeployment()), "nil event should fail")
}

// Create a set of fake deployments for testing.
func fakeDeploymentEvents() []*central.SensorEvent {
	return []*central.SensorEvent{
		{
			Resource: &central.SensorEvent_Deployment{
				Deployment: &storage.Deployment{
					Id: "id1",
					Containers: []*storage.Container{
						{
							Image: &storage.ContainerImage{
								Id: "sha1",
							},
						},
					},
				},
			},
			Action: central.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &central.SensorEvent_Deployment{
				Deployment: &storage.Deployment{
					Id: "id2",
					Containers: []*storage.Container{
						{
							Image: &storage.ContainerImage{
								Id: "sha1",
							},
						},
					},
				},
			},
			Action: central.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &central.SensorEvent_Deployment{
				Deployment: &storage.Deployment{
					Id: "id3",
					Containers: []*storage.Container{
						{
							Image: &storage.ContainerImage{
								Id: "sha2",
							},
						},
					},
				},
			},
			Action: central.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &central.SensorEvent_Deployment{
				Deployment: &storage.Deployment{
					Id: "id4",
					Containers: []*storage.Container{
						{
							Image: &storage.ContainerImage{
								Id: "sha2",
							},
						},
						{
							Image: &storage.ContainerImage{
								Id: "sha2",
							},
						},
					},
				},
			},
			Action: central.ResourceAction_CREATE_RESOURCE,
		},
	}
}
