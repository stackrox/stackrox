package expiringcache

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Cache implements a cache where the elements expire after a specific time
//
//go:generate mockgen-wrapper
type Cache[K comparable, V any] interface {
	Add(key K, value V)
	Get(key K) (V, bool)
	GetAll() []V
	GetOrSet(key K, value V) V
	Remove(key ...K)
	RemoveAll()
}

type opts[K comparable, V any] func(*expiringCacheImpl[K, V])

// UpdateExpirationOnGets resets the clock for a specific object when it is retrieved
func UpdateExpirationOnGets[K comparable, V any](e *expiringCacheImpl[K, V]) {
	e.updateOnGets = true
}

// NewExpiringCache returns a new lru Cache with time based expiration on values.
func NewExpiringCache[K comparable, V any](expiry time.Duration, options ...opts[K, V]) Cache[K, V] {
	return NewExpiringCacheWithClock[K, V](realClock{}, expiry, options...)
}

// NewExpiringCacheWithClock returns a new lru Cache with time based expiration on values, using the input clock.
func NewExpiringCacheWithClock[K comparable, V any](clock Clock, expiry time.Duration, options ...opts[K, V]) Cache[K, V] {
	e := &expiringCacheImpl[K, V]{
		mq: newMappedQueue(),

		clock:  clock,
		expiry: expiry,
	}
	for _, o := range options {
		o(e)
	}
	return e
}

type expiringCacheImpl[K comparable, V any] struct {
	lock sync.Mutex

	mq mappedQueue

	clock        Clock
	expiry       time.Duration
	updateOnGets bool

	latestScheduledPrune time.Time
}

// Add adds a new key/value pair to the cache.
func (e *expiringCacheImpl[K, V]) Add(key K, value V) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.addNoLock(key, value)
}

func (e *expiringCacheImpl[K, V]) addNoLock(key K, value interface{}) {
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
func (e *expiringCacheImpl[K, V]) Get(key K) (V, bool) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())
	value := e.getValue(key)
	if value == nil {
		var empty V
		return empty, false
	}

	if e.updateOnGets {
		e.removeNoLock(key)
		e.addNoLock(key, value)
	}
	return value.(V), true
}

// GetAll returns all non-expired values in the cache.
func (e *expiringCacheImpl[K, V]) GetAll() []V {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())

	cachedValues := e.mq.getAllValues()
	actualValues := make([]V, 0, len(cachedValues))
	for _, cv := range cachedValues {
		v, _ := unwrap[V](cv)
		actualValues = append(actualValues, v)
	}
	return actualValues
}

func (e *expiringCacheImpl[K, V]) removeNoLock(keys ...K) {
	for _, key := range keys {
		e.mq.remove(key)
	}
}

// Remove removes a key in the cache if present. Returns if it was present.
func (e *expiringCacheImpl[K, V]) Remove(keys ...K) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.cleanNoLock(e.clock.Now())
	e.removeNoLock(keys...)
}

// RemoveAll removes all values from the cache.
func (e *expiringCacheImpl[K, V]) RemoveAll() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.mq.removeAll()
}

// GetOrSet returns the value for the key if it exists or sets the value if it does not
// In the case of setting the value, it will return the passed value
func (e *expiringCacheImpl[K, V]) GetOrSet(key K, value V) V {
	e.lock.Lock()
	defer e.lock.Unlock()

	currValue := e.getValue(key)
	if currValue != nil {
		if e.updateOnGets {
			e.removeNoLock(key)
			e.addNoLock(key, currValue)
		}
		return currValue.(V)
	}

	e.addNoLock(key, value)
	return value
}

func (e *expiringCacheImpl[K, V]) getValue(key interface{}) interface{} {
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

func unwrap[V any](value interface{}) (V, time.Time) {
	cv := value.(*cacheValue)
	return cv.value.(V), cv.at
}

// Intermittent clean up of expired values.
// Called during Add or Get when needed.
////////////////////////////////////////

func (e *expiringCacheImpl[K, V]) cleanNoLock(at time.Time) {
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
