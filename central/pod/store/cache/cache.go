package cache

import (
	"context"

	"github.com/stackrox/rox/central/pod/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// NewCachedStore caches the pod store
func NewCachedStore(store store.Store) (store.Store, error) {
	impl := &cacheImpl{
		store: store,

		cache: make(map[string]*storage.Pod),
	}

	if err := impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	store store.Store

	cache map[string]*storage.Pod
	lock  sync.RWMutex
}

func (c *cacheImpl) addNoLock(pod *storage.Pod) {
	c.cache[pod.GetId()] = pod
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	ctx := sac.WithAllAccess(context.Background())
	return c.store.Walk(ctx, func(pod *storage.Pod) error {
		// No need to clone objects pulled directly from the store
		c.cache[pod.GetId()] = pod
		return nil
	})
}

func (c *cacheImpl) GetIDs(_ context.Context) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ids := make([]string, 0, len(c.cache))
	for id := range c.cache {
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *cacheImpl) Count(_ context.Context) (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache), nil
}

func (c *cacheImpl) Exists(_ context.Context, id string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cache[id]
	return ok, nil
}

func (c *cacheImpl) GetKeys() ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}
	return keys, nil
}

func (c *cacheImpl) Get(_ context.Context, id string) (*storage.Pod, bool, error) {
	c.lock.RLock()
	pod, ok := c.cache[id]
	c.lock.RUnlock()
	if !ok {
		return nil, false, nil
	}
	return pod.Clone(), true, nil
}

func (c *cacheImpl) GetMany(_ context.Context, ids []string) ([]*storage.Pod, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	pods := make([]*storage.Pod, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		pod, ok := c.cache[id]
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		pods = append(pods, pod.Clone())
	}
	return pods, missingIndices, nil
}

func (c *cacheImpl) Walk(_ context.Context, fn func(pod *storage.Pod) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, pod := range c.cache {
		if err := fn(pod.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) Upsert(ctx context.Context, pod *storage.Pod) error {
	if err := c.store.Upsert(ctx, pod); err != nil {
		return err
	}
	clonedPod := pod.Clone()
	c.lock.Lock()
	defer c.lock.Unlock()

	c.addNoLock(clonedPod)
	return nil
}

func (c *cacheImpl) Delete(ctx context.Context, id string) error {
	if err := c.store.Delete(ctx, id); err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.cache, id)

	return nil
}
