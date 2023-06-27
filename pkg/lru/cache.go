package lru

import (
	hashicorpLRU "github.com/hashicorp/golang-lru/v2"
)

type cacheImpl[K comparable, V any] struct {
	*hashicorpLRU.Cache[K, V]
}

func New[K comparable, V any](size int) (Cache[K, V], error) {
	cache, err := hashicorpLRU.New[K, V](size)
	return &cacheImpl[K, V]{Cache: cache}, err
}

func NewWithEvict[K comparable, V any](size int, onEvicted func(key K, value V)) (Cache[K, V], error) {
	cache, err := hashicorpLRU.NewWithEvict[K, V](size, onEvicted)
	return &cacheImpl[K, V]{Cache: cache}, err
}

// Close does nothing for this type of cache.
func (c *cacheImpl[K, V]) Close() {}
