package cache

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/pod/store/mocks"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestPodCache(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	pod1 := fixtures.GetPod()

	pod2 := fixtures.GetPod()
	pod2.Id = "id2"

	// Call Get and return pod1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().Get(ctx, pod1.GetId()).Return(pod1, true, nil)
	pod, exists, err := cacheStore.Get(ctx, pod1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod1, pod)

	baseStore.EXPECT().Upsert(ctx, pod1).Return(nil)
	assert.NoError(t, cacheStore.Upsert(ctx, pod1))
	baseStore.EXPECT().Upsert(ctx, pod2).Return(nil)
	assert.NoError(t, cacheStore.Upsert(ctx, pod2))

	// Don't expect this to hit the underlying store
	pod, exists, err = cacheStore.Get(ctx, pod1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod1, pod)

	pod, exists, err = cacheStore.Get(ctx, pod2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod2, pod)

	baseStore.EXPECT().Delete(ctx, pod1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.Delete(ctx, pod1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	pod, exists, err = cacheStore.Get(ctx, pod1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, pod)

	// Test acknowledgements
	baseStore.EXPECT().AckKeysIndexed(ctx, pod1.GetId(), pod2.GetId()).Return(nil)
	assert.NoError(t, cacheStore.AckKeysIndexed(ctx, pod1.GetId(), pod2.GetId()))
}
