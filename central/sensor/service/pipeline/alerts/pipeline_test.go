package alerts

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAlertsPipeline(t *testing.T) {
	suite.Run(t, new(AlertsPipelineTestSuite))
}

type AlertsPipelineTestSuite struct {
	suite.Suite

	clusters    *clusterMocks.MockDataStore
	deployments *deploymentMocks.MockDataStore
	manager     *lifecycleMocks.MockManager
	pipeline    *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *AlertsPipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.manager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.clusters, suite.deployments, suite.manager).(*pipelineImpl)
}

func (suite *AlertsPipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// buildRemoveMsg constructs a REMOVE_RESOURCE alert message for the given deployment ID.
func buildRemoveMsg(deploymentID string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: deploymentID,
					},
				},
			},
		},
	}
}

// TestDeploymentRemovedGuard_SkippedWhenFlagEnabled verifies that DeploymentRemoved is NOT
// called on the lifecycle manager when the DeploymentTombstones feature flag is enabled.
// In that case alert lifecycle is managed by the deployment events pipeline via
// DeploymentTombstoned, so calling DeploymentRemoved here would overwrite TOMBSTONED state.
func (suite *AlertsPipelineTestSuite) TestDeploymentRemovedGuard_SkippedWhenFlagEnabled() {
	suite.T().Setenv(features.DeploymentTombstones.EnvVar(), "true")

	const clusterID = "cluster-1"
	const deploymentID = "dep-1"

	suite.clusters.EXPECT().
		GetClusterName(gomock.Any(), clusterID).
		Return("test-cluster", true, nil)

	// DeploymentRemoved must NOT be called when the tombstone feature is active.
	suite.manager.EXPECT().DeploymentRemoved(gomock.Any()).Times(0)

	err := suite.pipeline.Run(context.Background(), clusterID, buildRemoveMsg(deploymentID), nil)
	suite.NoError(err)
}

// TestDeploymentRemovedGuard_CalledWhenFlagDisabled verifies that DeploymentRemoved IS
// called on the lifecycle manager when the DeploymentTombstones feature flag is disabled,
// preserving the original alert-cleanup behavior for hard-deleted deployments.
func (suite *AlertsPipelineTestSuite) TestDeploymentRemovedGuard_CalledWhenFlagDisabled() {
	suite.T().Setenv(features.DeploymentTombstones.EnvVar(), "false")

	const clusterID = "cluster-1"
	const deploymentID = "dep-1"

	suite.clusters.EXPECT().
		GetClusterName(gomock.Any(), clusterID).
		Return("test-cluster", true, nil)

	// DeploymentRemoved must be called exactly once when tombstoning is disabled.
	suite.manager.EXPECT().DeploymentRemoved(deploymentID).Return(nil)

	err := suite.pipeline.Run(context.Background(), clusterID, buildRemoveMsg(deploymentID), nil)
	suite.NoError(err)
}
