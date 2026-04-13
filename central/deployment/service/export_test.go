//go:build sql_integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockExportServer struct {
	v1.DeploymentService_ExportDeploymentsServer
	ctx          context.Context
	deployments  []*storage.Deployment
	sendCallsNum int
}

func (m *mockExportServer) Context() context.Context {
	return m.ctx
}

func (m *mockExportServer) Send(resp *v1.ExportDeploymentResponse) error {
	m.deployments = append(m.deployments, resp.GetDeployment())
	m.sendCallsNum++
	return nil
}

func TestExportDeployments_DefaultExcludesSoftDeleted(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment.
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Create soft-deleted deployment.
	now := time.Now()
	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	// Export with default parameters (include_deleted = false).
	req := &v1.ExportDeploymentRequest{
		Query:          "",
		IncludeDeleted: false,
	}

	mockServer := &mockExportServer{ctx: ctx}
	err = service.ExportDeployments(req, mockServer)
	require.NoError(t, err)

	// Verify only active deployment is returned.
	require.Len(t, mockServer.deployments, 1)
	assert.Equal(t, activeDeployment.GetId(), mockServer.deployments[0].GetId())
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, mockServer.deployments[0].GetLifecycleStage())
}

func TestExportDeployments_IncludeDeletedReturnsAll(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment.
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Create soft-deleted deployment.
	now := time.Now()
	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	// Export with include_deleted = true.
	req := &v1.ExportDeploymentRequest{
		Query:          "",
		IncludeDeleted: true,
	}

	mockServer := &mockExportServer{ctx: ctx}
	err = service.ExportDeployments(req, mockServer)
	require.NoError(t, err)

	// Verify both deployments are returned.
	require.Len(t, mockServer.deployments, 2)

	deploymentIDs := make(map[string]bool)
	for _, d := range mockServer.deployments {
		deploymentIDs[d.GetId()] = true
	}

	assert.True(t, deploymentIDs[activeDeployment.GetId()], "Active deployment should be included")
	assert.True(t, deploymentIDs[deletedDeployment.GetId()], "Deleted deployment should be included")
}

func TestExportDeployments_TombstoneFieldsSerialized(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create soft-deleted deployment with tombstone.
	now := time.Now()
	deletedAt := timestamppb.New(now.Add(-1 * time.Hour))
	expiresAt := timestamppb.New(now.Add(23 * time.Hour))

	deletedDeployment := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-deployment",
		ClusterId: uuid.NewV4().String(),
		Namespace: "default",
		Tombstone: &storage.Tombstone{
			DeletedAt: deletedAt,
			ExpiresAt: expiresAt,
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedDeployment))

	// Export with include_deleted = true.
	req := &v1.ExportDeploymentRequest{
		Query:          "",
		IncludeDeleted: true,
	}

	mockServer := &mockExportServer{ctx: ctx}
	err = service.ExportDeployments(req, mockServer)
	require.NoError(t, err)

	// Verify tombstone fields are serialized correctly.
	require.Len(t, mockServer.deployments, 1)
	exported := mockServer.deployments[0]

	assert.Equal(t, deletedDeployment.GetId(), exported.GetId())
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED, exported.GetLifecycleStage())

	tombstone := exported.GetTombstone()
	require.NotNil(t, tombstone, "Tombstone should be present in exported deployment")
	assert.NotNil(t, tombstone.GetDeletedAt(), "DeletedAt should be set")
	assert.NotNil(t, tombstone.GetExpiresAt(), "ExpiresAt should be set")

	// Verify timestamp values match.
	assert.Equal(t, deletedAt.AsTime().Unix(), tombstone.GetDeletedAt().AsTime().Unix())
	assert.Equal(t, expiresAt.AsTime().Unix(), tombstone.GetExpiresAt().AsTime().Unix())
}

func TestExportDeployments_ActiveDeploymentNoTombstone(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	// Create active deployment (no tombstone).
	activeDeployment := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-deployment",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      "default",
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
		Tombstone:      nil,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeDeployment))

	// Export with include_deleted = true.
	req := &v1.ExportDeploymentRequest{
		Query:          "",
		IncludeDeleted: true,
	}

	mockServer := &mockExportServer{ctx: ctx}
	err = service.ExportDeployments(req, mockServer)
	require.NoError(t, err)

	// Verify active deployment has no tombstone.
	require.Len(t, mockServer.deployments, 1)
	exported := mockServer.deployments[0]

	assert.Equal(t, activeDeployment.GetId(), exported.GetId())
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, exported.GetLifecycleStage())
	assert.Nil(t, exported.GetTombstone(), "Active deployment should not have tombstone")
}

func TestExportDeployments_QueryFilterCombinedWithLifecycleFilter(t *testing.T) {
	testDB := pgtest.ForT(t)
	defer testDB.Close()

	ctx := sac.WithAllAccess(context.Background())
	ds, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskManager := riskManagerMocks.NewMockManager(ctrl)

	service := New(ds, nil, nil, nil, nil, riskManager).(*serviceImpl)

	namespace1 := "namespace-1"
	namespace2 := "namespace-2"

	// Create active deployment in namespace-1.
	activeNs1 := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-ns1",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      namespace1,
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeNs1))

	// Create active deployment in namespace-2.
	activeNs2 := &storage.Deployment{
		Id:             uuid.NewV4().String(),
		Name:           "active-ns2",
		ClusterId:      uuid.NewV4().String(),
		Namespace:      namespace2,
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, activeNs2))

	// Create soft-deleted deployment in namespace-1.
	now := time.Now()
	deletedNs1 := &storage.Deployment{
		Id:        uuid.NewV4().String(),
		Name:      "deleted-ns1",
		ClusterId: uuid.NewV4().String(),
		Namespace: namespace1,
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-1 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(23 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	require.NoError(t, ds.UpsertDeployment(ctx, deletedNs1))

	// Export with namespace filter AND default include_deleted = false.
	req := &v1.ExportDeploymentRequest{
		Query:          "Namespace:" + namespace1,
		IncludeDeleted: false,
	}

	mockServer := &mockExportServer{ctx: ctx}
	err = service.ExportDeployments(req, mockServer)
	require.NoError(t, err)

	// Verify only active deployment in namespace-1 is returned.
	require.Len(t, mockServer.deployments, 1)
	assert.Equal(t, activeNs1.GetId(), mockServer.deployments[0].GetId())
	assert.Equal(t, namespace1, mockServer.deployments[0].GetNamespace())
	assert.Equal(t, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE, mockServer.deployments[0].GetLifecycleStage())
}
