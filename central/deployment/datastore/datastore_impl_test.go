package datastore

import (
	"context"
	"errors"
	"testing"

	storeMocks "github.com/stackrox/rox/central/deployment/datastore/internal/store/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	matcherMocks "github.com/stackrox/rox/central/platform/matcher/mocks"
	baselineMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
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
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestDeploymentDatastoreSuite(t *testing.T) {
	suite.Run(t, new(DeploymentDataStoreTestSuite))
}

type DeploymentDataStoreTestSuite struct {
	suite.Suite

	matcher   *matcherMocks.MockPlatformMatcher
	storage   *storeMocks.MockStore
	riskStore *riskMocks.MockDataStore
	filter    filter.Filter

	ctx context.Context

	mockCtrl *gomock.Controller
}

func (suite *DeploymentDataStoreTestSuite) SetupTest() {
	suite.ctx = sac.WithAllAccess(context.Background())

	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.riskStore = riskMocks.NewMockDataStore(mockCtrl)
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
			Name: "container-a",
			Image: &storage.ContainerImage{
				Id: "abc",
			},
		},
		{
			Name: "container-b",
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

	// Old deployment has fewer containers (e.g., no init containers yet), but matching containers still merge
	returnedDep := dep.CloneVT()
	returnedDep.Containers = returnedDep.GetContainers()[:1]
	returnedDep.Containers[0].Image.Id = "abc"

	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	// container-b still has no ID since it wasn't in the old deployment
	protoassert.Equal(suite.T(), expectedDep, dep)

	// Old container has matching name but different image full name, so no merge
	returnedDep = &storage.Deployment{
		Id:   "id",
		Type: kubernetes.CronJob,
		Containers: []*storage.Container{
			{
				Name: "container-a",
				Image: &storage.ContainerImage{
					Id: "abc",
				},
			},
			{
				Name: "container-b",
				Image: &storage.ContainerImage{
					Id: "xyz",
					Name: &storage.ImageName{
						FullName: "fullname",
					},
				},
			},
		},
	}
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	protoassert.Equal(suite.T(), expectedDep, dep)

	// Fill in missing last container value since image names match
	dep.Containers[1].Image.Name = &storage.ImageName{FullName: "fullname"}
	expectedDep.Containers[1].Image.Name = &storage.ImageName{FullName: "fullname"}
	expectedDep.Containers[1].Image.Id = "xyz"
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	if features.FlattenImageData.Enabled() {
		expectedDep.GetContainers()[1].GetImage().IdV2 = utils.NewImageV2ID(expectedDep.GetContainers()[1].GetImage().GetName(), expectedDep.GetContainers()[1].GetImage().GetId())
	}
	protoassert.Equal(suite.T(), expectedDep, dep)
}

func (suite *DeploymentDataStoreTestSuite) TestRemoveDeployment() {
	clusterID := "cluster-1"
	deploymentID := "dep-1"

	cases := map[string]struct {
		softDeletionEnabled bool
		// For soft-delete error paths.
		getErr    error
		getExists bool
		upsertErr error
		// For hard-delete error paths.
		deleteErr    error
		expectErr    bool
		expectUpsert bool
		expectDelete bool
	}{
		"hard delete when feature flag disabled": {
			softDeletionEnabled: false,
			expectDelete:        true,
		},
		"hard delete propagates Delete error": {
			softDeletionEnabled: false,
			deleteErr:           errors.New("delete error"),
			expectDelete:        true,
			expectErr:           true,
		},
		"soft delete when feature flag enabled": {
			softDeletionEnabled: true,
			getExists:           true,
			expectUpsert:        true,
		},
		"soft delete returns error when Get fails": {
			softDeletionEnabled: true,
			getErr:              errors.New("get error"),
			expectErr:           true,
		},
		"soft delete no-ops when deployment not found": {
			softDeletionEnabled: true,
			getExists:           false,
		},
		"soft delete propagates Upsert error": {
			softDeletionEnabled: true,
			getExists:           true,
			upsertErr:           errors.New("upsert error"),
			expectUpsert:        true,
			expectErr:           true,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			testutils.MustUpdateFeature(suite.T(), features.DeploymentSoftDeletion, tc.softDeletionEnabled)

			flowStoreMock := nfDSMocks.NewMockFlowDataStore(suite.mockCtrl)
			nfMock := nfDSMocks.NewMockClusterDataStore(suite.mockCtrl)
			baselinesMock := baselineMocks.NewMockDataStore(suite.mockCtrl)

			ds := newDatastoreImpl(suite.storage, nil, nil, baselinesMock, nfMock, suite.riskStore, nil, suite.filter, nil, nil, ranking.NewRanker(), suite.matcher)

			// Expect related object cleanup.
			suite.riskStore.EXPECT().RemoveRisk(gomock.Any(), deploymentID, storage.RiskSubjectType_DEPLOYMENT).Return(nil)
			baselinesMock.EXPECT().RemoveProcessBaselinesByDeployment(gomock.Any(), deploymentID).Return(nil)
			nfMock.EXPECT().GetFlowStore(gomock.Any(), clusterID).Return(flowStoreMock, nil)
			flowStoreMock.EXPECT().RemoveFlowsForDeployment(gomock.Any(), deploymentID).Return(nil)

			if tc.expectDelete {
				suite.storage.EXPECT().Delete(gomock.Any(), deploymentID).Return(tc.deleteErr)
			}

			if tc.softDeletionEnabled {
				deployment := &storage.Deployment{
					Id:        deploymentID,
					ClusterId: clusterID,
					State:     storage.DeploymentState_DEPLOYMENT_STATE_ACTIVE,
				}
				if tc.getErr != nil {
					suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(nil, false, tc.getErr)
				} else if !tc.getExists {
					suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(nil, false, nil)
				} else {
					suite.storage.EXPECT().Get(gomock.Any(), deploymentID).Return(deployment, true, nil)
				}

				if tc.expectUpsert {
					suite.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, dep *storage.Deployment) error {
							suite.Equal(storage.DeploymentState_DEPLOYMENT_STATE_DELETED, dep.GetState())
							suite.NotNil(dep.GetDeleted())
							return tc.upsertErr
						})
				}
			}

			err := ds.RemoveDeployment(suite.ctx, clusterID, deploymentID)
			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
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
