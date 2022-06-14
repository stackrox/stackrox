package cache

import (
	"context"

	"github.com/stackrox/stackrox/central/deployment/store"
	"github.com/stackrox/stackrox/central/deployment/store/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/size"
	"github.com/stackrox/stackrox/pkg/sizeboundedcache"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	maxCachedDeploymentSize = 50 * size.KB  // if it's larger than 50KB then we aren't going to cache it
	maxCacheSize            = 200 * size.MB // 200 MB
)

var (
	log = logging.LoggerForModule()
)

type deploymentTombstone struct{}

func sizeFunc(k, v interface{}) int64 {
	if dep, ok := v.(*storage.Deployment); ok {
		return int64(len(k.(string)) + dep.Size())
	}
	return int64(len(k.(string)))
}

// NewCachedStore returns an deployment store implementation that caches deployments
func NewCachedStore(store store.Store) store.Store {
	// This is a size based cache, where we use LRU to determine which of the oldest elements should
	// be removed to allow a new element
	cache, err := sizeboundedcache.New(maxCacheSize, maxCachedDeploymentSize, sizeFunc)
	utils.CrashOnError(err)

	return &cachedStore{
		store: store,
		cache: cache,
	}
}

// This cached store implementation relies on the usage of the Deployment store so this may not be easily portable
// to other sections of the code.
type cachedStore struct {
	store store.Store
	cache sizeboundedcache.Cache
}

func (c *cachedStore) testAndSetCacheEntry(id string, deployment *storage.Deployment) {
	c.cache.TestAndSet(id, deployment, func(_ interface{}, exists bool) bool {
		return !exists
	})
	c.updateStats()
}

func (c *cachedStore) getCachedDeployment(id string) (*storage.Deployment, bool, error) {
	entry, ok := c.cache.Get(id)
	if !ok {
		return nil, false, nil
	}
	if _, ok := entry.(*deploymentTombstone); ok {
		return nil, true, nil
	}
	return entry.(*storage.Deployment).Clone(), true, nil
}

func (c *cachedStore) GetListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	deployment, hadEntry, err := c.getCachedDeployment(id)
	if err != nil {
		return nil, false, err
	}
	if hadEntry {
		if deployment == nil {
			return nil, false, nil
		}
		return types.ConvertDeploymentToDeploymentList(deployment), true, nil
	}
	return c.store.GetListDeployment(ctx, id)
}

func (c *cachedStore) GetManyListDeployments(ctx context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	var deployments []*storage.ListDeployment
	var missingIndices []int
	for i, id := range ids {
		fullDeployment, hadEntry, err := c.getCachedDeployment(id)
		if err != nil {
			return nil, nil, err
		}
		if hadEntry {
			deploymentStoreCacheHits.Inc()
			// Tombstone entry existed
			if fullDeployment == nil {
				missingIndices = append(missingIndices, i)
			} else {
				deployments = append(deployments, types.ConvertDeploymentToDeploymentList(fullDeployment))
			}
			continue
		}
		deploymentStoreCacheMisses.Inc()

		listDeployment, exists, err := c.store.GetListDeployment(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			missingIndices = append(missingIndices, i)
			continue
		}
		deployments = append(deployments, listDeployment)
	}
	return deployments, missingIndices, nil
}

func (c *cachedStore) Get(ctx context.Context, id string) (*storage.Deployment, bool, error) {
	deployment, entryExists, err := c.getCachedDeployment(id)
	if err != nil {
		return nil, false, err
	}
	if entryExists {
		deploymentStoreCacheHits.Inc()
		// if entry is a tombstone entry, return that the deployment doesn't exist
		return deployment, deployment != nil, nil
	}

	deploymentStoreCacheMisses.Inc()
	deployment, exists, err := c.store.Get(ctx, id)
	if err != nil || !exists {
		return nil, exists, err
	}

	c.testAndSetCacheEntry(id, deployment)
	return deployment, true, nil
}

func (c *cachedStore) GetMany(ctx context.Context, ids []string) ([]*storage.Deployment, []int, error) {
	var deployments []*storage.Deployment
	var missingIndices []int
	for i, id := range ids {
		deployment, hadEntry, err := c.getCachedDeployment(id)
		if err != nil {
			return nil, nil, err
		}
		if hadEntry {
			deploymentStoreCacheHits.Inc()
			// Tombstone entry existed
			if deployment == nil {
				missingIndices = append(missingIndices, i)
			} else {
				deployments = append(deployments, deployment)
			}
			continue
		}
		deploymentStoreCacheMisses.Inc()

		deployment, exists, err := c.store.Get(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			missingIndices = append(missingIndices, i)
			continue
		}

		// Add the deployment to the cache
		c.testAndSetCacheEntry(id, deployment)
		deployments = append(deployments, deployment)
	}
	return deployments, missingIndices, nil
}

func (c *cachedStore) Count(ctx context.Context) (int, error) {
	return c.store.Count(ctx)
}

func (c *cachedStore) Upsert(ctx context.Context, deployment *storage.Deployment) error {
	if err := c.store.Upsert(ctx, deployment); err != nil {
		return err
	}

	c.cache.Add(deployment.GetId(), deployment)
	c.updateStats()
	return nil
}

func (c *cachedStore) Delete(ctx context.Context, id string) error {
	if err := c.store.Delete(ctx, id); err != nil {
		return err
	}
	c.cache.Add(id, &deploymentTombstone{})
	c.updateStats()
	return nil
}

func (c *cachedStore) updateStats() {
	objects, size := c.cache.Stats()
	deploymentStoreCacheObjects.Set(float64(objects))
	deploymentStoreCacheSize.Set(float64(size))
}

func (c *cachedStore) GetIDs(ctx context.Context) ([]string, error) {
	return c.store.GetIDs(ctx)
}
