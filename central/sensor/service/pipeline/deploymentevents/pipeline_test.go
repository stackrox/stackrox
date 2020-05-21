package deploymentevents

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	graphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	clusters       *clusterMocks.MockDataStore
	deployments    *deploymentMocks.MockDataStore
	images         *imageMocks.MockDataStore
	manager        *lifecycleMocks.MockManager
	graphEvaluator *graphMocks.MockEvaluator
	pipeline       *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.images = imageMocks.NewMockDataStore(suite.mockCtrl)
	suite.manager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.graphEvaluator = graphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.clusters, suite.deployments, suite.images, suite.manager, suite.graphEvaluator, nil).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestDeploymentRemovePipeline() {
	deployment := fixtures.GetDeployment()

	suite.deployments.EXPECT().RemoveDeployment(context.Background(), deployment.GetClusterId(), deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())

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

func (suite *PipelineTestSuite) TestAlertRemovalOnReconciliation() {
	deployment := fixtures.GetDeployment()

	suite.deployments.EXPECT().RemoveDeployment(context.Background(), deployment.GetClusterId(), deployment.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(deployment.GetClusterId())
	suite.manager.EXPECT().DeploymentRemoved(deployment)

	suite.NoError(suite.pipeline.runRemovePipeline(context.Background(), deployment, true))
}

func (suite *PipelineTestSuite) TestUpdateImages() {
	events := fakeDeploymentEvents()
	ctx := context.Background()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	suite.images.EXPECT().UpsertImage(ctx,
		testutils.PredMatcher("check that image has correct ID",
			func(img *storage.Image) bool { return img.GetId() == expectedImage0.GetId() })).Return(nil)

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(ctx, events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
}

func (suite *PipelineTestSuite) TestUpdateImagesSkipped() {
	ctx := context.Background()
	deployment := &storage.Deployment{
		Id: "id1",
		Containers: []*storage.Container{
			{
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "derp",
					},
				},
			},
		},
	}

	// Call function. It shouldn't do anything because the only image has no sha.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(ctx, deployment)
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
