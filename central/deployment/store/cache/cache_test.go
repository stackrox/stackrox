package cache

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/deployment/store/mocks"
	"github.com/stackrox/rox/central/deployment/store/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentCache(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	dep1 := fixtures.GetDeployment()
	listDep1 := types.ConvertDeploymentToDeploymentList(dep1)

	dep2 := fixtures.GetDeployment()
	dep2.Id = "id2"
	listDep2 := types.ConvertDeploymentToDeploymentList(dep2)

	baseStore.EXPECT().GetListDeployment(ctx, dep1.GetId()).Return(nil, false, nil)
	listDep, exists, err := cacheStore.GetListDeployment(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, listDep)

	// Call Get and return dep1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().Get(ctx, dep1.GetId()).Return(dep1, true, nil)
	dep, exists, err := cacheStore.Get(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, dep)

	baseStore.EXPECT().Upsert(ctx, dep1).Return(nil)
	assert.NoError(t, cacheStore.Upsert(ctx, dep1))
	baseStore.EXPECT().Upsert(ctx, dep2).Return(nil)
	assert.NoError(t, cacheStore.Upsert(ctx, dep2))

	// Don't expect this to hit the underlying store
	dep, exists, err = cacheStore.Get(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, dep)

	deployments, missing, err := cacheStore.GetMany(ctx, []string{dep1.GetId(), dep2.GetId()})
	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.Equal(t, deployments, []*storage.Deployment{dep1, dep2})

	baseStore.EXPECT().Get(ctx, "noid").Return(nil, false, nil)
	deployments, missing, err = cacheStore.GetMany(ctx, []string{dep1.GetId(), "noid", dep2.GetId()})
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, missing)
	assert.Equal(t, deployments, []*storage.Deployment{dep1, dep2})

	baseStore.EXPECT().GetListDeployment(ctx, "noid").Return(nil, false, nil)
	listDeployments, missing, err := cacheStore.GetManyListDeployments(ctx, dep1.GetId(), "noid", dep2.GetId())
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, missing)
	assert.Equal(t, listDeployments, []*storage.ListDeployment{listDep1, listDep2})

	listImage, exists, err := cacheStore.GetListDeployment(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertDeploymentToDeploymentList(dep1), listImage)

	listImage, exists, err = cacheStore.GetListDeployment(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertDeploymentToDeploymentList(dep1), listImage)

	dep, exists, err = cacheStore.Get(ctx, dep2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep2, dep)

	baseStore.EXPECT().Delete(ctx, dep1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.Delete(ctx, dep1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	dep, exists, err = cacheStore.Get(ctx, dep1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)
}
