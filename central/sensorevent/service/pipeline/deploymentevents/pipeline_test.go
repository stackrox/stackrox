package deploymentevents

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	ctx         context.Context
	clusters    *clusterMocks.MockDataStore
	deployments *deploymentMocks.MockDataStore
	images      *imageMocks.MockDataStore
	detector    *mockDetector

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.images = imageMocks.NewMockDataStore(suite.mockCtrl)
	suite.detector = &mockDetector{}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestCreateResponseForUpdate() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	suite.detector.On("DeploymentUpdated", events[0].GetDeployment()).
		Return("a1", storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, nil)

	// Call function.
	tested := &createResponseImpl{
		onUpdate: suite.detector.DeploymentUpdated,
		onRemove: suite.detector.DeploymentRemoved,
	}
	response := tested.do(events[0].GetDeployment(), events[0].GetAction())

	// Pull one more time to get nil
	suite.Equal(events[0].GetDeployment().GetId(), response.GetDeployment().GetDeploymentId())
	suite.Equal("a1", response.GetDeployment().GetAlertId())
	suite.detector.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestCreateResponseForRemove() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	suite.detector.On("DeploymentRemoved", events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &createResponseImpl{
		onUpdate: suite.detector.DeploymentUpdated,
		onRemove: suite.detector.DeploymentRemoved,
	}
	response := tested.do(events[0].GetDeployment(), central.ResourceAction_REMOVE_RESOURCE)

	// Should be no response for remove since no enforcement should be needed.
	suite.Nil(response)
	suite.detector.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentCreate() {
	events := fakeDeploymentEvents()
	events[0].Action = central.ResourceAction_CREATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.EXPECT().UpsertDeployment(events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].Action, events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
}

func (suite *PipelineTestSuite) TestPersistDeploymentUpdate() {
	events := fakeDeploymentEvents()
	events[0].Action = central.ResourceAction_UPDATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.EXPECT().UpsertDeployment(events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].Action, events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
}

func (suite *PipelineTestSuite) TestPersistDeploymentRemove() {
	events := fakeDeploymentEvents()
	events[0].Action = central.ResourceAction_REMOVE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.EXPECT().RemoveDeployment(events[0].GetDeployment().GetId()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].GetAction(), events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
}

func (suite *PipelineTestSuite) TestUpdateImages() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	suite.images.EXPECT().UpsertImage(expectedImage0).Return(nil)

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
}

func (suite *PipelineTestSuite) TestUpdateImagesSkipped() {
	deployment := &storage.Deployment{
		Id: "id1",
		Containers: []*storage.Container{
			{
				Image: &storage.Image{
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
	tested.do(deployment)
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
							Image: &storage.Image{
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
							Image: &storage.Image{
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
							Image: &storage.Image{
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
							Image: &storage.Image{
								Id: "sha2",
							},
						},
						{
							Image: &storage.Image{
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

// Mock detector for testing.
type mockDetector struct {
	mock.Mock
}

func (d *mockDetector) DeploymentUpdated(deployment *storage.Deployment) (alertID string, enforcement storage.EnforcementAction, err error) {
	args := d.Called(deployment)
	return args.Get(0).(string), args.Get(1).(storage.EnforcementAction), args.Error(2)
}

func (d *mockDetector) DeploymentRemoved(deployment *storage.Deployment) error {
	args := d.Called(deployment)
	return args.Error(0)
}
