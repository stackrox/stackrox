package postgres

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processbaseline/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

func NewCachedStore(db *pgxpool.Pool) store.Store {
	pgStore := New(db)

	cache := make(map[string]*storage.ProcessBaseline)
	err := pgStore.Walk(func(pb *storage.ProcessBaseline) error {
		cache[pb.GetId()] = pb
		return nil
	})
	if err != nil {
		panic(err)
	}
	return &cachedStore{
		store: pgStore,
		cache: cache,
	}
}

type cachedStore struct {
	store Store

	cacheLock sync.RWMutex
	cache     map[string]*storage.ProcessBaseline
}

func (c *cachedStore) Get(id string) (*storage.ProcessBaseline, bool, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	pb, ok := c.cache[id]
	if !ok {
		return nil, false, nil
	}
	return pb.Clone(), true, nil
}

func (c *cachedStore) GetMany(ids []string) (baselines []*storage.ProcessBaseline, missing []int, err error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	for idx, id := range ids {
		pb, ok := c.cache[id]
		if !ok {
			missing = append(missing, idx)
			continue
		}
		baselines = append(baselines, pb.Clone())
	}
	return baselines, missing, nil
}

func (c *cachedStore) Walk(fn func(baseline *storage.ProcessBaseline) error) error {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	for _, pb := range c.cache {
		if err := fn(pb); err != nil {
			return err
		}
	}
	return nil
}

func (c *cachedStore) Upsert(baseline *storage.ProcessBaseline) error {
	if err := c.store.Upsert(baseline); err != nil {
		return err
	}

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.cache[baseline.GetId()] = baseline
	return nil
}

func (c *cachedStore) Delete(id string) error {
	if err := c.store.Delete(id); err != nil {
		return err
	}
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	delete(c.cache, id)
	return nil
}
