package expiringcache

import (
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

// Cache implements a cache where the elements expire after a specific time
type Cache interface {
	Add(key, value interface{})
	Get(key interface{}) interface{}
	GetAll() []interface{}
	Remove(key interface{}) bool
	RemoveAll()
}

// NewExpiringCacheOrPanic returns a new lru Cache with time based expiration on values.
func NewExpiringCacheOrPanic(size int, expiry time.Duration, pruningInterval time.Duration) Cache {
	e := &expiringCacheImpl{
		mq: newMappedQueue(size),

		clock:           realClock{},
		expiry:          expiry,
		pruningInterval: pruningInterval,
	}
	return e
}

// NewExpiringCacheWithClockOrPanic returns a new lru Cache with time based expiration on values, using the input clock.
func NewExpiringCacheWithClockOrPanic(size int, clock Clock, expiry time.Duration, pruningInterval time.Duration) Cache {
	e := &expiringCacheImpl{
		mq: newMappedQueue(size),

		clock:           clock,
		expiry:          expiry,
		pruningInterval: pruningInterval,
	}
	return e
}

type expiringCacheImpl struct {
	lock sync.Mutex

	mq mappedQueue

	clock           Clock
	expiry          time.Duration
	pruningInterval time.Duration
	lastPrune       time.Time
}

// Add adds a new key/value pair to the cache.
func (e *expiringCacheImpl) Add(key, value interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()

	addTime := e.clock.Now()
	e.maybeCleanAsync(addTime)

	e.mq.push(key, wrap(value, addTime))
}

// Get takes in a key and returns nil if the item doesn't exist or if the item has expired
func (e *expiringCacheImpl) Get(key interface{}) interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	getTime := e.clock.Now()
	e.maybeCleanAsync(getTime)

	value, at := e.getValueAndTimeNoLock(key)
	if value != nil && getTime.Sub(at) > e.expiry {
		e.mq.remove(key)
		return nil
	}
	return value
}

// GetAll returns all non-expired values in the cache.
func (e *expiringCacheImpl) GetAll() []interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	getAllTime := e.clock.Now()
	e.cleanNoLock(getAllTime) // remove all expired values before getting the list.

	cachedValues := e.mq.getAllValues()
	actualValues := make([]interface{}, 0, len(cachedValues))
	for _, cv := range cachedValues {
		v, _ := unwrap(cv)
		actualValues = append(actualValues, v)
	}
	return actualValues
}

// Remove removes a key in the cache if present. Returns if it was present.
func (e *expiringCacheImpl) Remove(key interface{}) (wasPresent bool) {
	e.lock.Lock()
	defer e.lock.Unlock()

	removeTime := e.clock.Now()
	e.maybeCleanAsync(removeTime)

	value, at := e.getValueAndTimeNoLock(key)
	if value != nil {
		e.mq.remove(key)
		if removeTime.Sub(at) < e.expiry {
			wasPresent = true // only return true if it was present and unexpired.
		}
	}
	return
}

// RemoveAll removes all values from the cache.
func (e *expiringCacheImpl) RemoveAll() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.mq.removeAll()
}

func (e *expiringCacheImpl) getValueAndTimeNoLock(key interface{}) (interface{}, time.Time) {
	value := e.mq.get(key)
	if value == nil {
		return nil, time.Time{}
	}
	return unwrap(value)
}

// Store the value and the time of insertion in the mappedQueue.
type cacheValue struct {
	value interface{}
	at    time.Time
}

func wrap(value interface{}, at time.Time) *cacheValue {
	return &cacheValue{
		value: value,
		at:    at,
	}
}

func unwrap(value interface{}) (interface{}, time.Time) {
	cv := value.(*cacheValue)
	return cv.value, cv.at
}

// Intermittent clean up of expired values.
// Called during Add or Get when needed.
////////////////////////////////////////

func (e *expiringCacheImpl) maybeCleanAsync(at time.Time) {
	// If we reached the prune interval, flush expired items.
	if e.lastPrune.Sub(at) > e.pruningInterval {
		e.lastPrune = at
		go e.clean(at)
	}
}

func (e *expiringCacheImpl) clean(at time.Time) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(at)
}

func (e *expiringCacheImpl) cleanNoLock(at time.Time) {
	for {
		key, value := e.mq.front()
		if key == nil {
			break
		}
		cv := value.(*cacheValue)
		if at.Sub(cv.at) > e.expiry {
			e.mq.pop()
		} else {
			break
		}
	}
}
