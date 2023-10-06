package util

import (
	"context"
	"errors"
	"testing"

	sacHelperMocks "github.com/stackrox/rox/central/role/sachelper/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	errBroken = errors.New("broken")
)

func TestClusterIDFromNameOrID(t *testing.T) {
	ctx := context.Background()
	clusterSACHelper := sacHelperMocks.NewMockClusterSacHelper(gomock.NewController(t))

	t.Run("no cluster id returned when sachelper returns error", func(t *testing.T) {
		clusterSACHelper.EXPECT().GetClustersForPermissions(ctx, nil, nil).Return(nil, errBroken)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterSACHelper, "", nil)
		assert.Error(t, err)
		assert.Empty(t, clusterID)
	})

	clusters := []*v1.ScopeObject{
		{Id: "cluster1-id", Name: "cluster1-name"},
		{Id: "cluster2-id", Name: "cluster2-name"},
	}

	t.Run("id returned on id match", func(t *testing.T) {
		clusterSACHelper.EXPECT().GetClustersForPermissions(ctx, nil, nil).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterSACHelper, "cluster2-id", nil)
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})

	t.Run("id returned on name match", func(t *testing.T) {
		clusterSACHelper.EXPECT().GetClustersForPermissions(ctx, nil, nil).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterSACHelper, "cluster2-name", nil)
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})

	t.Run("id returned on id match when name matches another clusters id", func(t *testing.T) {
		clusters := []*v1.ScopeObject{
			{Id: "cluster1-id", Name: "cluster2-id"},
			{Id: "cluster2-id", Name: "cluster2-name"},
		}

		clusterSACHelper.EXPECT().GetClustersForPermissions(ctx, nil, nil).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterSACHelper, "cluster2-id", nil)
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})
}
