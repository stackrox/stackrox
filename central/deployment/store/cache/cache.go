package cache

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/deployment/store/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxCachedDeploymentSize = 50 * 1024 * 1024         // if it's larger than 50KB then we aren't going to cache it
	maxCacheSize            = 200 * 1024 * 1024 * 1024 // 200 MB
)

var (
	log = logging.LoggerForModule()
)

type deploymentTombstone struct{}

func sizeFunc(k, v interface{}) int64 {
	if img, ok := v.(*storage.Deployment); ok {
		return int64(len(k.(string)) + img.Size())
	}
	return int64(len(k.(string)))
}

// NewCachedStore returns an deployment store implementation that caches deployments
func NewCachedStore(store store.Store) store.Store {
	// This is a size based cache, where we use LRU to determine which of the oldest elements should
	// be removed to allow a new element
	cache, err := sizeboundedcache.New(maxCacheSize, maxCachedDeploymentSize, sizeFunc)
	utils.Must(err)

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
	return proto.Clone(entry.(*storage.Deployment)).(*storage.Deployment), true, nil
}

func (c *cachedStore) ListDeployment(id string) (*storage.ListDeployment, bool, error) {
	deployment, exists, err := c.GetDeployment(id)
	if err != nil || !exists {
		return nil, false, err
	}
	return types.ConvertDeploymentToDeploymentList(deployment), true, nil
}

func (c *cachedStore) ListDeployments() ([]*storage.ListDeployment, error) {
	return c.store.ListDeployments()
}

func (c *cachedStore) ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error) {
	deployments, missing, err := c.GetDeploymentsWithIDs(ids...)
	if err != nil {
		return nil, nil, err
	}
	listDeployments := make([]*storage.ListDeployment, 0, len(deployments))
	for _, d := range deployments {
		listDeployments = append(listDeployments, types.ConvertDeploymentToDeploymentList(d))
	}
	return listDeployments, missing, nil
}

func (c *cachedStore) GetDeployment(id string) (*storage.Deployment, bool, error) {
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
	deployment, exists, err := c.store.GetDeployment(id)
	if err != nil || !exists {
		return nil, exists, err
	}

	c.testAndSetCacheEntry(id, deployment)
	return deployment, true, nil
}

func (c *cachedStore) GetDeployments() ([]*storage.Deployment, error) {
	return c.store.GetDeployments()
}

func (c *cachedStore) GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error) {
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

		deployment, exists, err := c.store.GetDeployment(id)
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

func (c *cachedStore) CountDeployments() (int, error) {
	return c.store.CountDeployments()
}

func (c *cachedStore) UpsertDeployment(deployment *storage.Deployment) error {
	if err := c.store.UpsertDeployment(deployment); err != nil {
		return err
	}

	c.cache.Add(deployment.GetId(), deployment)
	c.updateStats()
	return nil
}

func (c *cachedStore) RemoveDeployment(id string) error {
	if err := c.store.RemoveDeployment(id); err != nil {
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

func (c *cachedStore) GetDeploymentIDs() ([]string, error) {
	return c.store.GetDeploymentIDs()
}

func (c *cachedStore) AckKeysIndexed(keys ...string) error {
	return c.store.AckKeysIndexed()
}

func (c *cachedStore) GetKeysToIndex() ([]string, error) {
	return c.store.GetKeysToIndex()
}
