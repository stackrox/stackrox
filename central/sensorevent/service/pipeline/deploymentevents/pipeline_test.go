package deploymentevents

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	ctx         context.Context
	clusters    *clusterMocks.DataStore
	deployments *deploymentMocks.DataStore
	images      *imageMocks.DataStore
	detector    *mockDetector
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.clusters = &clusterMocks.DataStore{}
	suite.deployments = &deploymentMocks.DataStore{}
	suite.images = &imageMocks.DataStore{}
	suite.detector = &mockDetector{}
}

func (suite *PipelineTestSuite) TestCreateResponseForUpdate() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	suite.detector.On("DeploymentUpdated", events[0].GetDeployment()).
		Return("a1", v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, nil)

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
	response := tested.do(events[0].GetDeployment(), v1.ResourceAction_REMOVE_RESOURCE)

	// Pull one more time to get nil
	suite.Equal(events[0].GetDeployment().GetId(), response.GetDeployment().GetDeploymentId())
	suite.Empty(response.GetDeployment().GetAlertId())
	suite.Equal(v1.EnforcementAction_UNSET_ENFORCEMENT, response.GetDeployment().GetEnforcement())
	suite.detector.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentCreate() {
	events := fakeDeploymentEvents()
	events[0].Action = v1.ResourceAction_CREATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.On("UpsertDeployment", events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].Action, events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
	suite.deployments.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentUpdate() {
	events := fakeDeploymentEvents()
	events[0].Action = v1.ResourceAction_UPDATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.On("UpsertDeployment", events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].Action, events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
	suite.deployments.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentRemove() {
	events := fakeDeploymentEvents()
	events[0].Action = v1.ResourceAction_REMOVE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.On("RemoveDeployment", events[0].GetDeployment().GetId()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0].GetAction(), events[0].GetDeployment())

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
	suite.deployments.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestUpdateImages() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	suite.images.On("UpsertDedupeImage", expectedImage0).Return(nil)

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
	suite.images.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestUpdateImagesSkipped() {
	deployment := &v1.Deployment{
		Id: "id1",
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
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

	// Pull one more time to get nil
	suite.images.AssertExpectations(suite.T())
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
func fakeDeploymentEvents() []*v1.SensorEvent {
	return []*v1.SensorEvent{
		{
			Resource: &v1.SensorEvent_Deployment{
				Deployment: &v1.Deployment{
					Id: "id1",
					Containers: []*v1.Container{
						{
							Image: &v1.Image{
								Name: &v1.ImageName{
									Sha: "sha1",
								},
							},
						},
					},
				},
			},
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &v1.SensorEvent_Deployment{
				Deployment: &v1.Deployment{
					Id: "id2",
					Containers: []*v1.Container{
						{
							Image: &v1.Image{
								Name: &v1.ImageName{
									Sha: "sha1",
								},
							},
						},
					},
				},
			},
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &v1.SensorEvent_Deployment{
				Deployment: &v1.Deployment{
					Id: "id3",
					Containers: []*v1.Container{
						{
							Image: &v1.Image{
								Name: &v1.ImageName{
									Sha: "sha2",
								},
							},
						},
					},
				},
			},
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
			Resource: &v1.SensorEvent_Deployment{
				Deployment: &v1.Deployment{
					Id: "id4",
					Containers: []*v1.Container{
						{
							Image: &v1.Image{
								Name: &v1.ImageName{
									Sha: "sha3",
								},
							},
						},
						{
							Image: &v1.Image{
								Name: &v1.ImageName{
									Sha: "sha2",
								},
							},
						},
					},
				},
			},
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
	}
}

// Mock detector for testing.
type mockDetector struct {
	mock.Mock
}

func (d *mockDetector) DeploymentUpdated(deployment *v1.Deployment) (alertID string, enforcement v1.EnforcementAction, err error) {
	args := d.Called(deployment)
	return args.Get(0).(string), args.Get(1).(v1.EnforcementAction), args.Error(2)
}

func (d *mockDetector) DeploymentRemoved(deployment *v1.Deployment) error {
	args := d.Called(deployment)
	return args.Error(0)
}
