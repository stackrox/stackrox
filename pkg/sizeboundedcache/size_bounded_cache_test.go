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
