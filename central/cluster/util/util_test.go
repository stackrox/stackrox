package util

import (
	"context"
	"errors"
	"testing"

	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	errBroken = errors.New("broken")
)

func TestClusterIDFromNameOrID(t *testing.T) {
	ctx := context.Background()
	clusterDS := clusterDSMocks.NewMockDataStore(gomock.NewController(t))
	// fakeClusterName := "fake-cluster-name"
	// fakeClusterID := "fake-cluster-id"

	t.Run("no cluster id returned when datasource returns error", func(t *testing.T) {
		clusterDS.EXPECT().GetClusters(ctx).Return(nil, errBroken)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterDS, "")
		assert.Error(t, err)
		assert.Empty(t, clusterID)
	})

	clusters := []*storage.Cluster{
		{Id: "cluster1-id", Name: "cluster1-name"},
		{Id: "cluster2-id", Name: "cluster2-name"},
	}

	t.Run("id returned on id match", func(t *testing.T) {
		clusterDS.EXPECT().GetClusters(ctx).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterDS, "cluster2-id")
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})

	t.Run("id returned on name match", func(t *testing.T) {
		clusterDS.EXPECT().GetClusters(ctx).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterDS, "cluster2-name")
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})

	t.Run("id returned on id match when name matches another clusters id", func(t *testing.T) {
		clusters := []*storage.Cluster{
			{Id: "cluster1-id", Name: "cluster2-id"},
			{Id: "cluster2-id", Name: "cluster2-name"},
		}

		clusterDS.EXPECT().GetClusters(ctx).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterDS, "cluster2-id")
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})
}
