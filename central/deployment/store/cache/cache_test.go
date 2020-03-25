package cache

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/deployment/store/mocks"
	"github.com/stackrox/rox/central/deployment/store/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	dep1 := fixtures.GetDeployment()

	dep2 := fixtures.GetDeployment()
	dep2.Id = "id2"

	// Call Get and return dep1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().GetDeployment(dep1.GetId()).Return(dep1, true, nil)
	img, exists, err := cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, img)

	baseStore.EXPECT().UpsertDeployment(dep1).Return(nil)
	assert.NoError(t, cacheStore.UpsertDeployment(dep1))
	baseStore.EXPECT().UpsertDeployment(dep2).Return(nil)
	assert.NoError(t, cacheStore.UpsertDeployment(dep2))

	// Don't expect this to hit the underlying store
	img, exists, err = cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, img)

	listImage, exists, err := cacheStore.ListDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertDeploymentToDeploymentList(dep1), listImage)

	img, exists, err = cacheStore.GetDeployment(dep2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep2, img)

	baseStore.EXPECT().RemoveDeployment(dep1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.RemoveDeployment(dep1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	img, exists, err = cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, img)
}
