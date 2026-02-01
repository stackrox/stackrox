package sizeboundedcache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSizeFunc(k string, v string) int64 {
	return int64(len(k) + len(v))
}

func TestSizeBoundedCacheErrors(t *testing.T) {
	var cases = []struct {
		maxSize, maxItemSize int64
		costFunc             func(k string, v string) int64
	}{
		{
			maxSize:     0,
			maxItemSize: 10,
			costFunc:    testSizeFunc,
		},
		{
			maxSize:     10,
			maxItemSize: 0,
			costFunc:    testSizeFunc,
		},
		{
			maxSize:     10,
			maxItemSize: 10,
			costFunc:    nil,
		},
		{
			maxSize:     10,
			maxItemSize: 10,
			costFunc:    nil,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d-%d-%v", c.maxSize, c.maxItemSize, c.costFunc == nil), func(t *testing.T) {
			_, err := New(c.maxSize, c.maxItemSize, c.costFunc)
			assert.Error(t, err)
		})
	}
}

func TestSizeBoundedCache(t *testing.T) {
	cache, err := New(10, 9, testSizeFunc)
	require.NoError(t, err)
	cacheImpl := cache.(*sizeBoundedCache[string, string])

	// simple add
	cacheImpl.Add("a", "b")
	assert.Equal(t, int64(2), cacheImpl.currSize)
	assert.Equal(t, 1, cacheImpl.cache.Len())
	value, ok := cacheImpl.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "b", value)

	// duplicate add should do nothing
	cacheImpl.Add("a", "b")
	assert.Equal(t, int64(2), cacheImpl.currSize)
	assert.Equal(t, 1, cacheImpl.cache.Len())

	// simple replace should equal cost of key + cost of value
	cacheImpl.Add("a", "bcd")
	assert.Equal(t, int64(4), cacheImpl.currSize)
	assert.Equal(t, 1, cacheImpl.cache.Len())

	// Adding new value should increment
	cacheImpl.Add("b", "bcd")
	assert.Equal(t, int64(8), cacheImpl.currSize)
	assert.Equal(t, 2, cacheImpl.cache.Len())

	// Going from 8 -> 12 should evict a
	cacheImpl.Add("c", "bcd")
	assert.Equal(t, int64(8), cacheImpl.currSize)

	// "a" should have been evicted
	_, ok = cacheImpl.Get("a")
	assert.False(t, ok)

	// Add an entry that is too large
	cacheImpl.Add("aa", "bbbbbbbbbbbbbbbbb")
	assert.Equal(t, int64(8), cacheImpl.currSize)
	_, ok = cacheImpl.Get("aa")
	assert.False(t, ok)

	// add an entry that should evict multiple previous entries
	cacheImpl.Add("aa", "bbbbbbb")
	assert.Equal(t, int64(9), cacheImpl.currSize)
	_, ok = cacheImpl.Get("b")
	assert.False(t, ok)
	_, ok = cacheImpl.Get("c")
	assert.False(t, ok)

	cacheImpl.Remove("aa")
	_, ok = cacheImpl.Get("aa")
	assert.False(t, ok)
}

func TestSizeBoundedCacheResize(t *testing.T) {
	t.Run("resize to larger size preserves entries", func(t *testing.T) {
		cache, err := New[string, string](10, 5, testSizeFunc)
		require.NoError(t, err)
		cacheImpl := cache.(*sizeBoundedCache[string, string])

		// Add some entries
		cacheImpl.Add("a", "b") // size 2
		cacheImpl.Add("c", "d") // size 2
		assert.Equal(t, int64(4), cacheImpl.currSize)
		assert.Equal(t, 2, cacheImpl.cache.Len())

		// Resize to larger
		cache.Resize(20)
		assert.Equal(t, int64(20), cacheImpl.maxSize)

		// Entries should still be there
		assert.Equal(t, int64(4), cacheImpl.currSize)
		assert.Equal(t, 2, cacheImpl.cache.Len())
		_, ok := cacheImpl.Get("a")
		assert.True(t, ok)
		_, ok = cacheImpl.Get("c")
		assert.True(t, ok)
	})

	t.Run("resize to smaller size evicts entries", func(t *testing.T) {
		cache, err := New[string, string](20, 5, testSizeFunc)
		require.NoError(t, err)
		cacheImpl := cache.(*sizeBoundedCache[string, string])

		// Add entries to fill cache
		cacheImpl.Add("a", "bb") // size 3, oldest
		cacheImpl.Add("c", "dd") // size 3
		cacheImpl.Add("e", "ff") // size 3, newest
		assert.Equal(t, int64(9), cacheImpl.currSize)
		assert.Equal(t, 3, cacheImpl.cache.Len())

		// Resize to smaller - should evict oldest entries
		cache.Resize(7)
		assert.Equal(t, int64(7), cacheImpl.maxSize)

		// Should have evicted "a" (oldest) to fit within new limit
		assert.LessOrEqual(t, cacheImpl.currSize, int64(7))
		_, ok := cacheImpl.Get("a")
		assert.False(t, ok, "oldest entry should be evicted")

		// Newer entries should still be accessible
		_, ok = cacheImpl.Get("e")
		assert.True(t, ok, "newest entry should be preserved")
	})

	t.Run("resize to size smaller than maxItemSize is rejected", func(t *testing.T) {
		cache, err := New[string, string](20, 10, testSizeFunc)
		require.NoError(t, err)
		cacheImpl := cache.(*sizeBoundedCache[string, string])

		cacheImpl.Add("a", "b") // size 2
		originalMaxSize := cacheImpl.maxSize

		// Try to resize to less than maxItemSize (10)
		cache.Resize(5)

		// Size should remain unchanged
		assert.Equal(t, originalMaxSize, cacheImpl.maxSize)
	})

	t.Run("resize with empty cache", func(t *testing.T) {
		cache, err := New[string, string](10, 5, testSizeFunc)
		require.NoError(t, err)
		cacheImpl := cache.(*sizeBoundedCache[string, string])

		// Resize empty cache
		cache.Resize(20)
		assert.Equal(t, int64(20), cacheImpl.maxSize)
		assert.Equal(t, int64(0), cacheImpl.currSize)

		// Should still work after resize
		cacheImpl.Add("a", "b")
		_, ok := cacheImpl.Get("a")
		assert.True(t, ok)
	})
}
