//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetClustersForSACReflectsExternalChanges(t *testing.T) {
	db := pgtest.ForT(t)
	reader := &datastoreImpl{clusterStorage: pgStore.New(db.DB)}
	writer := pgStore.New(db.DB)
	ctx := sac.WithAllAccess(context.Background())
	cluster := &storage.Cluster{Id: uuid.NewV4().String(), Name: "cluster", Labels: map[string]string{"team": "old"}}

	clusters, err := reader.GetClustersForSAC(context.Background())
	require.NoError(t, err)
	require.Empty(t, clusters)

	// A second store instance models a write from another Central process.
	require.NoError(t, writer.Upsert(ctx, cluster))
	clusters, err = reader.GetClustersForSAC(context.Background())
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	require.Equal(t, cluster.GetId(), clusters[0].GetId())
	require.Equal(t, cluster.GetLabels(), clusters[0].GetLabels())

	cluster.Labels["team"] = "new"
	require.NoError(t, writer.Upsert(ctx, cluster))
	clusters, err = reader.GetClustersForSAC(context.Background())
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	require.Equal(t, cluster.GetLabels(), clusters[0].GetLabels())

	require.NoError(t, writer.Delete(ctx, cluster.GetId()))
	clusters, err = reader.GetClustersForSAC(context.Background())
	require.NoError(t, err)
	require.Empty(t, clusters)
}

func TestGetClustersForSACPropagatesCancellation(t *testing.T) {
	db := pgtest.ForT(t)
	reader := &datastoreImpl{clusterStorage: pgStore.New(db.DB)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	clusters, err := reader.GetClustersForSAC(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, clusters)
}
