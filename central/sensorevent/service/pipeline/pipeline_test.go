package pipeline

import (
	"context"
	"fmt"
	"testing"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	ctx         context.Context
	clusters    *clusterDataStore.MockDataStore
	deployments *deploymentDataStore.MockDataStore
	images      *imageDataStore.MockDataStore
	detector    *mockDetector
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.clusters = &clusterDataStore.MockDataStore{}
	suite.deployments = &deploymentDataStore.MockDataStore{}
	suite.images = &imageDataStore.MockDataStore{}
	suite.detector = &mockDetector{}
}

func (suite *PipelineTestSuite) TestCreateResponse() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	suite.detector.On("ProcessDeploymentEvent", events[0].GetDeployment(), events[0].GetAction()).Return("a1", v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)

	// Call function.
	tested := &createResponseImpl{
		toEnforcement: suite.detector.ProcessDeploymentEvent,
	}
	response := tested.do(events[0])

	// Pull one more time to get nil
	suite.Equal(events[0].GetDeployment().GetId(), response.GetDeploymentId())
	suite.Equal("a1", response.GetAlertId())
	suite.detector.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentCreate() {
	events := fakeDeploymentEvents()
	events[0].Action = v1.ResourceAction_CREATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.On("UpdateDeployment", events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0])

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
	suite.deployments.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistDeploymentUpdate() {
	events := fakeDeploymentEvents()
	events[0].Action = v1.ResourceAction_UPDATE_RESOURCE

	// Expect that our enforcement generator is called with expected data.
	suite.deployments.On("UpdateDeployment", events[0].GetDeployment()).Return(nil)

	// Call function.
	tested := &persistDeploymentImpl{
		deployments: suite.deployments,
	}
	err := tested.do(events[0])

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
	err := tested.do(events[0])

	// Pull one more time to get nil
	suite.NoError(err, "persistence should have succeeded")
	suite.deployments.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestPersistImages() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	suite.images.On("UpdateImage", expectedImage0).Return(nil)

	// Call function.
	tested := &persistImagesImpl{
		images: suite.images,
	}
	tested.do(events[0])

	// Pull one more time to get nil
	suite.images.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestUpdateImages() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := &v1.Image{}
	expectedImageSha0 := events[0].GetDeployment().GetContainers()[0].GetImage().GetName().GetSha()
	suite.images.On("GetImage", expectedImageSha0).Return(expectedImage0, true, nil)

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
	suite.images.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestUpdateImagesNoUpdate() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	expectedImageSha0 := events[0].GetDeployment().GetContainers()[0].GetImage().GetName().GetSha()
	suite.images.On("GetImage", expectedImageSha0).Return(expectedImage0, false, nil)

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
	suite.images.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestUpdateImagesError() {
	events := fakeDeploymentEvents()

	// Expect that our enforcement generator is called with expected data.
	expectedImage0 := events[0].GetDeployment().GetContainers()[0].GetImage()
	expectedImageSha0 := events[0].GetDeployment().GetContainers()[0].GetImage().GetName().GetSha()
	suite.images.On("GetImage", expectedImageSha0).Return(expectedImage0, false, fmt.Errorf("oh noes"))

	// Call function.
	tested := &updateImagesImpl{
		images: suite.images,
	}
	tested.do(events[0].GetDeployment())

	// Pull one more time to get nil
	suite.Equal(expectedImage0, events[0].GetDeployment().GetContainers()[0].GetImage())
	suite.images.AssertExpectations(suite.T())
}

func (suite *PipelineTestSuite) TestValidateImages() {
	events := fakeDeploymentEvents()

	// Call function.
	tested := &validateInputImpl{}

	// Pull one more time to get nil
	suite.NoError(tested.do(events[0]), "valid input should not throw an error.")

	// Pull one more time to get nil
	events[0].Deployment = nil
	suite.Error(tested.do(events[0]), "event without deployment should fail")

	// Pull one more time to get nil
	events[0] = nil
	suite.Error(tested.do(events[0]), "nil event should fail")
}

// Create a set of fake deployments for testing.
func fakeDeploymentEvents() []*v1.DeploymentEvent {
	return []*v1.DeploymentEvent{
		{
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
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
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
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
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
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
		{
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
			Action: v1.ResourceAction_CREATE_RESOURCE,
		},
	}
}

// Mock detector for testing.
type mockDetector struct {
	mock.Mock
}

func (d *mockDetector) ProcessDeploymentEvent(deployment *v1.Deployment, action v1.ResourceAction) (alertID string, enforcement v1.EnforcementAction) {
	args := d.Called(deployment, action)
	return args.Get(0).(string), args.Get(1).(v1.EnforcementAction)
}
