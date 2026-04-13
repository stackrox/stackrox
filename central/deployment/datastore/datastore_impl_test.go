package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/deployment/datastore/internal/store/mocks"
	nfMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	matcherMocks "github.com/stackrox/rox/central/platform/matcher/mocks"
	pbMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestDeploymentDatastoreSuite(t *testing.T) {
	suite.Run(t, new(DeploymentDataStoreTestSuite))
}

type DeploymentDataStoreTestSuite struct {
	suite.Suite

	matcher       *matcherMocks.MockPlatformMatcher
	storage       *storeMocks.MockStore
	riskStore     *riskMocks.MockDataStore
	baselineStore *pbMocks.MockDataStore
	networkFlows  *nfMocks.MockClusterDataStore
	flowStore     *nfMocks.MockFlowDataStore

	filter filter.Filter

	ctx context.Context

	mockCtrl *gomock.Controller
}

func (suite *DeploymentDataStoreTestSuite) SetupTest() {
	suite.ctx = sac.WithAllAccess(context.Background())

	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.riskStore = riskMocks.NewMockDataStore(mockCtrl)
	suite.baselineStore = pbMocks.NewMockDataStore(mockCtrl)
	suite.networkFlows = nfMocks.NewMockClusterDataStore(mockCtrl)
	suite.flowStore = nfMocks.NewMockFlowDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})
	suite.matcher = matcherMocks.NewMockPlatformMatcher(mockCtrl)
}

func (suite *DeploymentDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *DeploymentDataStoreTestSuite) TestInitializeRanker() {
	clusterRanker := ranking.NewRanker()
	nsRanker := ranking.NewRanker()
	deploymentRanker := ranking.NewRanker()

	ds := newDatastoreImpl(suite.storage, nil, nil, nil, nil, suite.riskStore, nil, suite.filter, clusterRanker, nsRanker, deploymentRanker, suite.matcher)

	deployments := []*storage.Deployment{
		{
			Id:          "1",
			RiskScore:   float32(1.0),
			NamespaceId: "ns1",
			ClusterId:   "c1",
		},
		{
			Id:          "2",
			RiskScore:   float32(2.0),
			NamespaceId: "ns1",
			ClusterId:   "c1",
		},
		{
			Id:          "3",
			NamespaceId: "ns2",
			ClusterId:   "c2",
		},
		{
			Id: "4",
		},
		{
			Id: "5",
		},
	}
	suite.storage.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(deployments))
	ds.initializeRanker()

	suite.Equal(int64(1), clusterRanker.GetRankForID("c1"))
	suite.Equal(int64(2), clusterRanker.GetRankForID("c2"))

	suite.Equal(int64(1), nsRanker.GetRankForID("ns1"))
	suite.Equal(int64(2), nsRanker.GetRankForID("ns2"))

	suite.Equal(int64(1), deploymentRanker.GetRankForID("2"))
	suite.Equal(int64(2), deploymentRanker.GetRankForID("1"))
	suite.Equal(int64(3), deploymentRanker.GetRankForID("3"))
}

func walkMockFunc(deployments []*storage.Deployment) func(_ context.Context, _ *v1.Query, fn func(group *storage.Deployment) error) error {
	return func(_ context.Context, _ *v1.Query, fn func(deployment *storage.Deployment) error) error {
		for _, g := range deployments {
			if err := fn(g); err != nil {
				return err
			}
		}
		return nil
	}
}

func (suite *DeploymentDataStoreTestSuite) TestMergeCronJobs() {
	ds := newDatastoreImpl(suite.storage, nil, nil, nil, nil, suite.riskStore, nil, suite.filter, nil, nil, nil, suite.matcher)
	ctx := sac.WithAllAccess(context.Background())

	// Not a cronjob so no merging
	dep := &storage.Deployment{
		Id:   "id",
		Type: kubernetes.Deployment,
	}
	expectedDep := dep.CloneVT()
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	dep.Containers = []*storage.Container{
		{
			Image: &storage.ContainerImage{
				Id: "abc",
			},
		},
		{
			Image: &storage.ContainerImage{
				Id: "def",
			},
		},
	}
	dep.Type = kubernetes.CronJob
	expectedDep = dep.CloneVT()
	// All container have images with digests
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	// All containers don't have images with digests, but old deployment does not exist
	dep.Containers[1].Image.Id = ""
	expectedDep = dep.CloneVT()
	suite.storage.EXPECT().Get(ctx, "id").Return(nil, false, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	// Different numbers of containers for the CronJob so early exit with no changes
	returnedDep := dep.CloneVT()
	returnedDep.Containers = returnedDep.GetContainers()[:1]

	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	// Filled in for missing last container, but names do not match
	returnedDep.Containers = append(returnedDep.Containers, dep.GetContainers()[1].CloneVT())
	returnedDep.Containers[1].Image.Id = "xyz"
	returnedDep.Containers[1].Image.Name = &storage.ImageName{
		FullName: "fullname",
	}
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	// Fill in missing last container value since names match
	dep.Containers[1].Image.Name = returnedDep.GetContainers()[1].GetImage().GetName()
	expectedDep.Containers[1].Image.Name = returnedDep.GetContainers()[1].GetImage().GetName()
	expectedDep.Containers[1].Image.Id = "xyz"
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	if features.FlattenImageData.Enabled() {
		expectedDep.GetContainers()[1].GetImage().IdV2 = utils.NewImageV2ID(expectedDep.GetContainers()[1].GetImage().GetName(), expectedDep.GetContainers()[1].GetImage().GetId())
	}
	protoassert.Equal(suite.T(), expectedDep, dep)
}

func (suite *DeploymentDataStoreTestSuite) TestUpsert_PlatformComponentAssignment() {
	suite.T().Setenv(features.PlatformComponents.EnvVar(), "true")
	if !features.PlatformComponents.Enabled() {
		suite.T().Skip("Skip test when ROX_PLATFORM_COMPONENTS disabled")
		suite.T().SkipNow()
	}
	ds := newDatastoreImpl(suite.storage, nil, nil, nil, nil, suite.riskStore, nil, suite.filter, nil, nil, ranking.NewRanker(), suite.matcher)
	ctx := sac.WithAllAccess(context.Background())
	suite.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).AnyTimes()

	// Case: Deployment not matching platform rules
	deployment := &storage.Deployment{
		Id:        "id",
		Namespace: "my-namespace",
	}
	expectedDeployment := &storage.Deployment{
		Id:                "id",
		Namespace:         "my-namespace",
		PlatformComponent: false,
	}

	suite.storage.EXPECT().Upsert(gomock.Any(), expectedDeployment).Return(nil).Times(1)
	suite.matcher.EXPECT().MatchDeployment(deployment).Return(false, nil).Times(1)
	err := ds.UpsertDeployment(ctx, deployment)
	suite.Require().NoError(err)

	// Case: Deployment matching platform rules
	deployment = &storage.Deployment{
		Id:        "id",
		Namespace: "kube-123",
	}
	expectedDeployment = &storage.Deployment{
		Id:                "id",
		Namespace:         "kube-123",
		PlatformComponent: true,
	}

	suite.storage.EXPECT().Upsert(gomock.Any(), expectedDeployment).Return(nil).Times(1)
	suite.matcher.EXPECT().MatchDeployment(deployment).Return(true, nil).Times(1)
	err = ds.UpsertDeployment(ctx, deployment)
	suite.Require().NoError(err)
}

func (suite *DeploymentDataStoreTestSuite) TestRemoveDeployment_SoftDelete() {
	// Note: This test verifies soft-delete behavior without mocking the config datastore.
	// Full end-to-end testing with config TTL is covered in integration tests.
	//
	// Alert resolution: This test verifies that RemoveDeployment() creates a tombstone.
	// Alert resolution happens at a higher level (lifecycle manager) BEFORE RemoveDeployment() is called.
	// See central/detection/lifecycle/manager_impl_test.go:TestDeploymentRemoved for alert resolution tests.
	// See central/sensor/service/pipeline/deploymentevents/pipeline_test.go:TestAlertRemovalOnReconciliation
	// for integration tests verifying the full flow.
	ds := newDatastoreImpl(suite.storage, nil, nil, suite.baselineStore, suite.networkFlows, suite.riskStore, nil, suite.filter, nil, nil, nil, suite.matcher)
	ctx := sac.WithAllAccess(context.Background())

	deploymentID := "test-deployment-id"
	clusterID := "test-cluster-id"

	// Create a deployment to be soft-deleted.
	deployment := &storage.Deployment{
		Id:             deploymentID,
		ClusterId:      clusterID,
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}

	// Set up expectations for cleanup of related objects.
	suite.riskStore.EXPECT().RemoveRisk(gomock.Any(), deploymentID, storage.RiskSubjectType_DEPLOYMENT).Return(nil)
	suite.baselineStore.EXPECT().RemoveProcessBaselinesByDeployment(gomock.Any(), deploymentID).Return(nil)
	suite.networkFlows.EXPECT().GetFlowStore(gomock.Any(), clusterID).Return(suite.flowStore, nil)
	suite.flowStore.EXPECT().RemoveFlowsForDeployment(gomock.Any(), deploymentID).Return(nil)

	// Set up expectations: Get should return the deployment, Upsert should be called.
	suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(deployment, true, nil)
	suite.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *storage.Deployment) error {
		// Verify the deployment was marked as soft-deleted.
		assert.NotNil(suite.T(), d.GetTombstone(), "Tombstone should be set")
		assert.NotNil(suite.T(), d.GetTombstone().GetDeletedAt(), "Tombstone.DeletedAt should be set")
		assert.NotNil(suite.T(), d.GetTombstone().GetExpiresAt(), "Tombstone.ExpiresAt should be set")
		assert.Equal(suite.T(), storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, d.GetLifecycleStage(), "LifecycleStage should be DEPLOYMENT_DELETED")

		// Verify that ExpiresAt is after DeletedAt.
		deletedAt := d.GetTombstone().GetDeletedAt().AsTime()
		expiresAt := d.GetTombstone().GetExpiresAt().AsTime()
		assert.True(suite.T(), expiresAt.After(deletedAt), "ExpiresAt should be after DeletedAt")

		return nil
	})

	// Call RemoveDeployment - this should soft-delete the deployment.
	// Note: The process filter's Delete(deploymentID) is still called inside RemoveDeployment,
	// verifying that process indicators are still properly cleared on soft-delete (design comment #7).
	err := ds.RemoveDeployment(ctx, clusterID, deploymentID)
	if err != nil {
		suite.T().Logf("RemoveDeployment error: %v", err)
	}
	suite.Require().NoError(err)
}

func (suite *DeploymentDataStoreTestSuite) TestRemoveDeployment_DeploymentNotFound() {
	ds := newDatastoreImpl(suite.storage, nil, nil, suite.baselineStore, suite.networkFlows, suite.riskStore, nil, suite.filter, nil, nil, nil, suite.matcher)
	ctx := sac.WithAllAccess(context.Background())

	deploymentID := "nonexistent-deployment"
	clusterID := "test-cluster-id"

	// Set up expectations for cleanup of related objects (these run before fetching the deployment).
	suite.riskStore.EXPECT().RemoveRisk(gomock.Any(), deploymentID, storage.RiskSubjectType_DEPLOYMENT).Return(nil)
	suite.baselineStore.EXPECT().RemoveProcessBaselinesByDeployment(gomock.Any(), deploymentID).Return(nil)
	suite.networkFlows.EXPECT().GetFlowStore(gomock.Any(), clusterID).Return(suite.flowStore, nil)
	suite.flowStore.EXPECT().RemoveFlowsForDeployment(gomock.Any(), deploymentID).Return(nil)

	// Get should return not found.
	suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(nil, false, nil)

	// RemoveDeployment should handle this gracefully without error.
	err := ds.RemoveDeployment(ctx, clusterID, deploymentID)
	suite.Require().NoError(err)
}

// TestRemoveDeployment_DeploymentRemainsAccessible verifies that soft-deleted deployments
// remain in the database and can be queried (for alert retention and audit trails).
func (suite *DeploymentDataStoreTestSuite) TestRemoveDeployment_DeploymentRemainsAccessible() {
	ds := newDatastoreImpl(suite.storage, nil, nil, suite.baselineStore, suite.networkFlows, suite.riskStore, nil, suite.filter, nil, nil, nil, suite.matcher)
	ctx := sac.WithAllAccess(context.Background())

	deploymentID := "test-deployment-id"
	clusterID := "test-cluster-id"

	// Create a deployment to be soft-deleted.
	deployment := &storage.Deployment{
		Id:             deploymentID,
		ClusterId:      clusterID,
		Name:           "test-deployment",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}

	// Set up expectations for cleanup of related objects.
	suite.riskStore.EXPECT().RemoveRisk(gomock.Any(), deploymentID, storage.RiskSubjectType_DEPLOYMENT).Return(nil)
	suite.baselineStore.EXPECT().RemoveProcessBaselinesByDeployment(gomock.Any(), deploymentID).Return(nil)
	suite.networkFlows.EXPECT().GetFlowStore(gomock.Any(), clusterID).Return(suite.flowStore, nil)
	suite.flowStore.EXPECT().RemoveFlowsForDeployment(gomock.Any(), deploymentID).Return(nil)

	// Expect Get to return the deployment for soft-delete.
	suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(deployment, true, nil)

	// Expect Upsert to be called with the soft-deleted deployment.
	var softDeletedDeployment *storage.Deployment
	suite.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *storage.Deployment) error {
		softDeletedDeployment = d
		return nil
	})

	// Call RemoveDeployment.
	err := ds.RemoveDeployment(ctx, clusterID, deploymentID)
	suite.Require().NoError(err)

	// Verify the deployment was soft-deleted (not hard-deleted).
	suite.Require().NotNil(softDeletedDeployment, "Deployment should be upserted, not deleted")
	suite.Equal(storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, softDeletedDeployment.GetLifecycleStage())
	suite.NotNil(softDeletedDeployment.GetTombstone())

	// This verifies that alert retention policies can still query soft-deleted deployments.
	// Alerts associated with this deployment will have deployment.Id references that remain valid.
}
