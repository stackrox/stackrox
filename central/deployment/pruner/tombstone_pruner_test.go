package pruner

import (
	"testing"
	"time"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTombstonePruner_PrunesExpiredDeployments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	// Create expired deployments (expires_at in the past).
	now := time.Now()
	expiredDeployment1 := &storage.Deployment{
		Id:        "expired-1",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}
	expiredDeployment2 := &storage.Deployment{
		Id:        "expired-2",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-48 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-24 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}

	// Mock GetExpiredDeployments to return the two expired deployments.
	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{
		expiredDeployment1,
		expiredDeployment2,
	}, nil).Times(1)

	// Mock RemoveDeployment for both.
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired-1").Return(nil).Times(1)
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired-2").Return(nil).Times(1)

	pruner := NewTombstonePruner(deploymentDS).(*tombstonePrunerImpl)
	pruner.pruneExpiredDeployments()

	// Verify that both deployments were pruned.
	assert.Equal(t, uint64(2), pruner.GetPrunedTotal())
}

func TestTombstonePruner_DoesNotPruneNonExpiredDeployments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	// Mock GetExpiredDeployments to return empty (no expired deployments).
	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{}, nil).Times(1)

	pruner := NewTombstonePruner(deploymentDS).(*tombstonePrunerImpl)
	pruner.pruneExpiredDeployments()

	// Verify no deployments were pruned.
	assert.Equal(t, uint64(0), pruner.GetPrunedTotal())
}

func TestTombstonePruner_HandlesRemoveDeploymentError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	now := time.Now()
	expiredDeployment := &storage.Deployment{
		Id:        "expired",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)),
		},
		LifecycleStage: storage.DeploymentLifecycleStage_DEPLOYMENT_DELETED,
	}

	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{expiredDeployment}, nil).Times(1)
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired").Return(assert.AnError).Times(1)

	pruner := NewTombstonePruner(deploymentDS).(*tombstonePrunerImpl)
	pruner.pruneExpiredDeployments()

	// Verify that the pruner logged the error but did not panic.
	// The pruned count should be 0 since the removal failed.
	assert.Equal(t, uint64(0), pruner.GetPrunedTotal())
}

func TestTombstonePruner_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	// The pruner runs immediately on Start, so expect at least one call.
	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{}, nil).AnyTimes()

	pruner := NewTombstonePruner(deploymentDS)
	pruner.Start()

	// Let it run for a short time.
	time.Sleep(100 * time.Millisecond)

	// Stop the pruner.
	pruner.Stop()

	// Verify that stop completed without hanging.
	// The test will timeout if Stop() blocks indefinitely.
}

func TestTombstonePruner_UpdatesLastPruneTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{}, nil).Times(1)

	pruner := NewTombstonePruner(deploymentDS).(*tombstonePrunerImpl)
	initialTime := pruner.GetLastPruneTime()
	require.True(t, initialTime.IsZero(), "Initial last prune time should be zero")

	pruner.pruneExpiredDeployments()

	updatedTime := pruner.GetLastPruneTime()
	require.False(t, updatedTime.IsZero(), "Last prune time should be updated after pruning")
	require.True(t, updatedTime.After(initialTime), "Last prune time should be after the initial time")
}

func TestTombstonePruner_AccumulatesPrunedTotal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentDS := deploymentMocks.NewMockDataStore(ctrl)

	now := protocompat.TimestampNow().AsTime()

	// First prune cycle: 2 deployments.
	expired1 := &storage.Deployment{
		Id:        "expired-1",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)),
		},
	}
	expired2 := &storage.Deployment{
		Id:        "expired-2",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)),
		},
	}

	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{expired1, expired2}, nil).Times(1)
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired-1").Return(nil).Times(1)
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired-2").Return(nil).Times(1)

	pruner := NewTombstonePruner(deploymentDS).(*tombstonePrunerImpl)
	pruner.pruneExpiredDeployments()
	assert.Equal(t, uint64(2), pruner.GetPrunedTotal())

	// Second prune cycle: 1 deployment.
	expired3 := &storage.Deployment{
		Id:        "expired-3",
		ClusterId: "cluster-1",
		Tombstone: &storage.Tombstone{
			DeletedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			ExpiresAt: timestamppb.New(now.Add(-1 * time.Hour)),
		},
	}

	deploymentDS.EXPECT().GetExpiredDeployments(gomock.Any()).Return([]*storage.Deployment{expired3}, nil).Times(1)
	deploymentDS.EXPECT().RemoveDeployment(gomock.Any(), "cluster-1", "expired-3").Return(nil).Times(1)

	pruner.pruneExpiredDeployments()
	assert.Equal(t, uint64(3), pruner.GetPrunedTotal(), "Total should accumulate across prune cycles")
}
