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

	so := &v1.ScopeObject{}
	so.SetId("cluster1-id")
	so.SetName("cluster1-name")
	so2 := &v1.ScopeObject{}
	so2.SetId("cluster2-id")
	so2.SetName("cluster2-name")
	clusters := []*v1.ScopeObject{
		so,
		so2,
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
		so3 := &v1.ScopeObject{}
		so3.SetId("cluster1-id")
		so3.SetName("cluster2-id")
		so4 := &v1.ScopeObject{}
		so4.SetId("cluster2-id")
		so4.SetName("cluster2-name")
		clusters := []*v1.ScopeObject{
			so3,
			so4,
		}

		clusterSACHelper.EXPECT().GetClustersForPermissions(ctx, nil, nil).Return(clusters, nil)
		clusterID, err := GetClusterIDFromNameOrID(ctx, clusterSACHelper, "cluster2-id", nil)
		assert.NoError(t, err)
		assert.Equal(t, "cluster2-id", clusterID)
	})
}
