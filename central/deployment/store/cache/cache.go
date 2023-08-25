package cache

import (
	"context"

	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/deployment/store/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// NewCachedStore caches the deployment store
func NewCachedStore(store store.Store) (store.Store, error) {
	impl := &cacheImpl{
		store: store,

		cache: make(map[string]*storage.Deployment),
	}

	if err := impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	store store.Store

	cache map[string]*storage.Deployment
	lock  sync.RWMutex
}

func (c *cacheImpl) GetListDeployment(_ context.Context, id string) (*storage.ListDeployment, bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	deployment, ok := c.cache[id]
	if !ok {
		return nil, false, nil
	}
	return types.ConvertDeploymentToDeploymentList(deployment), true, nil
}

func (c *cacheImpl) GetManyListDeployments(_ context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	listDeployments := make([]*storage.ListDeployment, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		deployment, ok := c.cache[id]
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		listDeployments = append(listDeployments, types.ConvertDeploymentToDeploymentList(deployment))
	}
	return listDeployments, missingIndices, nil
}

func (c *cacheImpl) addNoLock(deployment *storage.Deployment) {
	c.cache[deployment.GetId()] = deployment
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	ctx := sac.WithAllAccess(context.Background())
	return c.store.Walk(ctx, func(d *storage.Deployment) error {
		c.cache[d.GetId()] = d
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

func (c *cacheImpl) Get(_ context.Context, id string) (*storage.Deployment, bool, error) {
	c.lock.RLock()
	deployment, ok := c.cache[id]
	c.lock.RUnlock()
	if !ok {
		return nil, false, nil
	}
	return deployment.Clone(), true, nil
}

func (c *cacheImpl) GetMany(_ context.Context, ids []string) ([]*storage.Deployment, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	deployments := make([]*storage.Deployment, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		deployment, ok := c.cache[id]
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		deployments = append(deployments, deployment.Clone())
	}
	return deployments, missingIndices, nil
}

func (c *cacheImpl) Walk(_ context.Context, fn func(deployment *storage.Deployment) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, deployment := range c.cache {
		if err := fn(deployment.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) Upsert(ctx context.Context, deployment *storage.Deployment) error {
	if err := c.store.Upsert(ctx, deployment); err != nil {
		return err
	}
	clonedDeployment := deployment.Clone()
	c.lock.Lock()
	defer c.lock.Unlock()

	c.addNoLock(clonedDeployment)
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
