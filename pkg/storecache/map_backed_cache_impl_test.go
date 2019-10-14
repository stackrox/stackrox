package storecache

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMapBackedCache(t *testing.T) {
	suite.Run(t, new(mapBackedCacheTestSuite))
}

type mapBackedCacheTestSuite struct {
	suite.Suite
}

func (suite *mapBackedCacheTestSuite) TestAddGetRemove() {
	cache := NewMapBackedCache()
	key := "key"
	version := uint64(1337)

	gotten := cache.Get(key)
	suite.Nil(gotten)

	element := "cache element"
	cache.Add(key, element, version)
	gotten = cache.Get(key)
	suite.Equal(element, gotten)

	cache.Add(key, "old cache element", version-1)
	gotten = cache.Get(key)
	suite.Equal(element, gotten)

	removed := cache.Remove("Not a key", version)
	suite.False(removed)
	removed = cache.Remove(key, version)
	suite.True(removed)
	gotten = cache.Get(key)
	suite.Nil(gotten)

	cache.Add(key, element, version-1)
	gotten = cache.Get(key)
	suite.Nil(gotten)
}
