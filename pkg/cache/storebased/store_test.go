package storebased

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetObject(t *testing.T) {
	var callCount int
	fn := func(ctx context.Context, id string) (*storage.ServiceAccount, error) {
		m := map[string]*storage.ServiceAccount{
			"1": {Name: "1"},
			"2": {Name: "2"},
			"3": {Name: "3"},
		}
		callCount++
		return m[id], nil
	}

	cache := NewCache(fn)

	obj, err := cache.GetObject(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, &storage.ServiceAccount{Name: "1"}, obj)
	assert.Equal(t, 1, callCount)

	obj2, err := cache.GetObject(context.Background(), "2")
	assert.NoError(t, err)
	assert.Equal(t, &storage.ServiceAccount{Name: "2"}, obj2)
	assert.Equal(t, 2, callCount)

	assert.Equal(t, cache.cachedObjects.GetMap(), map[string]*storage.ServiceAccount{"1": obj, "2": obj2})

	obj, err = cache.GetObject(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, &storage.ServiceAccount{Name: "1"}, obj)
	assert.Equal(t, 2, callCount)

	obj2, err = cache.GetObject(context.Background(), "2")
	assert.NoError(t, err)
	assert.Equal(t, &storage.ServiceAccount{Name: "2"}, obj2)
	assert.Equal(t, 2, callCount)
}

func TestInvalidateCache(t *testing.T) {
	cache := NewCache[*storage.ServiceAccount](nil)

	cache.cachedObjects.SetMany(map[string]*storage.ServiceAccount{
		"1": {Name: "1"},
		"2": {Name: "2"},
		"3": {Name: "3"},
		"4": {Name: "4"},
	})

	cache.InvalidateCache("1", "2", "3")

	assert.Equal(t, cache.cachedObjects.GetMap(), map[string]*storage.ServiceAccount{"4": {Name: "4"}})
}
