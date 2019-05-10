package expiringcache

import (
	"container/list"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

type elem struct {
	key    interface{}
	data   interface{}
	expiry time.Time
}

// Cache implements a cache where the elements expire after a specific time
type Cache interface {
	Add(key, value interface{})
	Get(key interface{}) interface{}
	GetAll() []interface{}
	Remove(key interface{}) bool
	RemoveAll()
}

type expiringCacheImpl struct {
	lock sync.RWMutex

	size  int
	queue *list.List
	items map[interface{}]*list.Element

	expiry          time.Duration
	pruningInterval time.Duration
}

// NewExpiringCacheOrPanic returns a new Cache and only panics if the size is negative
// per the lru.Cache spec
func NewExpiringCacheOrPanic(size int, expiry time.Duration, pruningInterval time.Duration) Cache {
	e := &expiringCacheImpl{
		size:            size,
		expiry:          expiry,
		pruningInterval: pruningInterval,

		queue: list.New(),
		items: make(map[interface{}]*list.Element),
	}
	if pruningInterval > 0 {
		go e.start()
	}
	return e
}

func (e *expiringCacheImpl) start() {
	t := time.NewTicker(e.pruningInterval)
	for range t.C {
		e.clean()
	}
}

func (e *expiringCacheImpl) clean() {
	e.lock.Lock()
	defer e.lock.Unlock()

	for {
		front := e.queue.Front()
		if front == nil {
			break
		}
		if front.Value.(*elem).expiry.Before(time.Now()) {
			e.removeNoLock(e.queue.Front())
		} else {
			break
		}
	}
}

func (e *expiringCacheImpl) Add(key, value interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()

	// The closest one to expiring will be removed first
	if e.queue.Len() == e.size {
		// Front has the oldest
		e.removeNoLock(e.queue.Front())
	}
	listElement := e.queue.PushBack(&elem{
		key:    key,
		data:   value,
		expiry: time.Now().Add(e.expiry)})
	e.items[key] = listElement
}

func (e *expiringCacheImpl) getNoLock(k interface{}) interface{} {
	listElem, ok := e.items[k]
	if !ok {
		return nil
	}
	element := listElem.Value.(*elem)
	if element.expiry.Before(time.Now()) {
		e.removeNoLock(listElem)
		return nil
	}
	return element.data
}

// Get takes in a key and returns nil if the item doesn't exist or if the item has expired
func (e *expiringCacheImpl) Get(k interface{}) interface{} {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.getNoLock(k)
}

func (e *expiringCacheImpl) GetAll() []interface{} {
	e.lock.RLock()
	defer e.lock.RUnlock()

	ret := make([]interface{}, 0, len(e.items))
	for k := range e.items {
		if val := e.getNoLock(k); val != nil {
			ret = append(ret, val)
		}
	}
	return ret
}

func (e *expiringCacheImpl) removeNoLock(deleteElement *list.Element) {
	e.queue.Remove(deleteElement)
	delete(e.items, deleteElement.Value.(*elem).key)
}

func (e *expiringCacheImpl) Remove(key interface{}) bool {
	e.lock.Lock()
	defer e.lock.Unlock()
	value, ok := e.items[key]
	if !ok {
		return false
	}
	e.removeNoLock(value)
	return true
}

func (e *expiringCacheImpl) RemoveAll() {
	e.lock.Lock()
	defer e.lock.Unlock()

	for e.queue.Front() != nil {
		e.removeNoLock(e.queue.Front())
	}
}
