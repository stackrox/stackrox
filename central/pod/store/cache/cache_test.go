package cache

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/pod/store/mocks"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestPodCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	pod1 := fixtures.GetPod()

	pod2 := fixtures.GetPod()
	pod2.Id = "id2"

	// Call Get and return pod1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().Get(pod1.GetId()).Return(pod1, true, nil)
	pod, exists, err := cacheStore.Get(pod1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod1, pod)

	baseStore.EXPECT().Upsert(pod1).Return(nil)
	assert.NoError(t, cacheStore.Upsert(pod1))
	baseStore.EXPECT().Upsert(pod2).Return(nil)
	assert.NoError(t, cacheStore.Upsert(pod2))

	// Don't expect this to hit the underlying store
	pod, exists, err = cacheStore.Get(pod1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod1, pod)

	pod, exists, err = cacheStore.Get(pod2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, pod2, pod)

	baseStore.EXPECT().Delete(pod1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.Delete(pod1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	pod, exists, err = cacheStore.Get(pod1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, pod)

	// Test acknowledgements
	baseStore.EXPECT().AckKeysIndexed(pod1.GetId(), pod2.GetId()).Return(nil)
	assert.NoError(t, cacheStore.AckKeysIndexed(pod1.GetId(), pod2.GetId()))
}
