package expiringcache

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Cache implements a cache where the elements expire after a specific time
//
//go:generate mockgen-wrapper
type Cache interface {
	Add(key, value interface{})
	Get(key interface{}) interface{}
	GetAll() []interface{}
	GetOrSet(key interface{}, value interface{}) interface{}
	Remove(key ...interface{})
	RemoveAll()
}

type opts func(*expiringCacheImpl)

// UpdateExpirationOnGets resets the clock for a specific object when it is retrieved
func UpdateExpirationOnGets(e *expiringCacheImpl) {
	e.updateOnGets = true
}

// NewExpiringCache returns a new lru Cache with time based expiration on values.
func NewExpiringCache(expiry time.Duration, options ...opts) Cache {
	return NewExpiringCacheWithClock(realClock{}, expiry, options...)
}

// NewExpiringCacheWithClock returns a new lru Cache with time based expiration on values, using the input clock.
func NewExpiringCacheWithClock(clock Clock, expiry time.Duration, options ...opts) Cache {
	e := &expiringCacheImpl{
		mq: newMappedQueue(),

		clock:  clock,
		expiry: expiry,
	}
	for _, o := range options {
		o(e)
	}
	return e
}

type expiringCacheImpl struct {
	lock sync.Mutex

	mq mappedQueue

	clock        Clock
	expiry       time.Duration
	updateOnGets bool

	latestScheduledPrune time.Time
}

// Add adds a new key/value pair to the cache.
func (e *expiringCacheImpl) Add(key, value interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.addNoLock(key, value)
}

func (e *expiringCacheImpl) addNoLock(key, value interface{}) {
	now := e.clock.Now()
	e.cleanNoLock(now)

	// Look at scheduled runs
	var newTime time.Time

	// Expiry time plus padding of 1 second
	interval := e.expiry + 1*time.Second

	// If there is no next scheduled prune time or it's in the past, then the next scheduled time
	// is Now + interval
	if e.latestScheduledPrune.IsZero() || e.latestScheduledPrune.Before(now) {
		newTime = now.Add(interval)
	} else if e.latestScheduledPrune.Before(now.Add(interval)) {
		// there is a scheduled prune, but it's not after now + interval then schedule one from the next prune + interval
		newTime = e.latestScheduledPrune.Add(interval)
	}
	// The final case is that there is a prune scheduled after now.Add(interval) which will automatically clean up the value being added
	if !newTime.IsZero() {
		e.latestScheduledPrune = newTime
		go func(sleepDuration time.Duration) {
			time.Sleep(sleepDuration)
			concurrency.WithLock(&e.lock, func() {
				e.cleanNoLock(time.Now())
			})
		}(newTime.Sub(now))
	}

	e.mq.push(key, wrap(value, now))
}

// Get takes in a key and returns nil if the item doesn't exist or if the item has expired
func (e *expiringCacheImpl) Get(key interface{}) interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())
	value := e.getValue(key)
	if value == nil {
		return nil
	}

	if e.updateOnGets {
		e.removeNoLock(key)
		e.addNoLock(key, value)
	}
	return value
}

// GetAll returns all non-expired values in the cache.
func (e *expiringCacheImpl) GetAll() []interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())

	cachedValues := e.mq.getAllValues()
	actualValues := make([]interface{}, 0, len(cachedValues))
	for _, cv := range cachedValues {
		v, _ := unwrap(cv)
		actualValues = append(actualValues, v)
	}
	return actualValues
}

func (e *expiringCacheImpl) removeNoLock(keys ...interface{}) {
	for _, key := range keys {
		e.mq.remove(key)
	}
}

// Remove removes a key in the cache if present. Returns if it was present.
func (e *expiringCacheImpl) Remove(keys ...interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())
	e.removeNoLock(keys...)
}

// RemoveAll removes all values from the cache.
func (e *expiringCacheImpl) RemoveAll() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.mq.removeAll()
}

// GetOrSet returns the value for the key if it exists or sets the value if it does not
// In the case of setting the value, it will return the passed value
func (e *expiringCacheImpl) GetOrSet(key, value interface{}) interface{} {
	e.lock.Lock()
	defer e.lock.Unlock()

	currValue := e.getValue(key)
	if currValue != nil {
		if e.updateOnGets {
			e.removeNoLock(key)
			e.addNoLock(key, currValue)
		}
		return currValue
	}

	e.addNoLock(key, value)
	return value
}

func (e *expiringCacheImpl) getValue(key interface{}) interface{} {
	value := e.mq.get(key)
	if value == nil {
		return nil
	}
	cv := value.(*cacheValue)
	return cv.value
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
