package storecache

import "github.com/stackrox/rox/pkg/sync"

type mapBackedCacheImpl struct {
	mapLock            sync.RWMutex
	backingMap         map[interface{}]*cacheValue
	lastRemovedVersion uint64
}

type cacheValue struct {
	value   interface{}
	version uint64
}

// NewMapBackedCache creates and returns a mapBackedCacheImpl
func NewMapBackedCache() Cache {
	return &mapBackedCacheImpl{
		backingMap: make(map[interface{}]*cacheValue),
	}
}

// Add adds a value to the cache
func (m *mapBackedCacheImpl) Add(key, value interface{}, version uint64) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	oldVersion := m.lastRemovedVersion
	oldValue, ok := m.backingMap[key]
	if ok {
		oldVersion = oldValue.version
	}
	if version < oldVersion {
		return
	}
	m.backingMap[key] = &cacheValue{
		value,
		version,
	}
}

// Get returns a value from the cache
func (m *mapBackedCacheImpl) Get(key interface{}) interface{} {
	m.mapLock.RLock()
	defer m.mapLock.RUnlock()
	if value, ok := m.backingMap[key]; ok {
		return value.value
	}
	return nil
}

// Remove removes a value from the cache
func (m *mapBackedCacheImpl) Remove(key interface{}, version uint64) bool {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	startingSize := len(m.backingMap)
	if m.lastRemovedVersion < version {
		m.lastRemovedVersion = version
	}
	delete(m.backingMap, key)
	return startingSize > len(m.backingMap)
}
