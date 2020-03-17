package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	podSearch "github.com/stackrox/rox/central/pod/datastore/internal/search"
	podIndex "github.com/stackrox/rox/central/pod/index"
	podStore "github.com/stackrox/rox/central/pod/store"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	podBatchSize = 1000
)

var (
	// It should not be possible that pod and deployment scope are different,
	// so just use the same access controls as a deployment.
	podsSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	podStore    podStore.Store
	podIndexer  podIndex.Indexer
	podSearcher podSearch.Searcher

	indicators    piDS.DataStore
	processFilter filter.Filter

	keyedMutex *concurrency.KeyedMutex
}

func newDatastoreImpl(storage podStore.Store, indexer podIndex.Indexer, searcher podSearch.Searcher,
	indicators piDS.DataStore, processFilter filter.Filter) (*datastoreImpl, error) {
	ds := &datastoreImpl{
		podStore:      storage,
		podIndexer:    indexer,
		podSearcher:   searcher,
		indicators:    indicators,
		processFilter: processFilter,
		keyedMutex:    concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (ds *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()

	needsReindexing, err := ds.podIndexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsReindexing {
		return ds.fullReindex()
	}

	log.Info("[STARTUP] Determining if pod db/indexer reconciliation is needed")

	podsToIndex, err := ds.podStore.GetKeysToIndex()
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index")
	}

	log.Infof("[STARTUP] Found %d Pods to index", len(podsToIndex))

	podBatcher := batcher.New(len(podsToIndex), podBatchSize)
	for start, end, valid := podBatcher.Next(); valid; start, end, valid = podBatcher.Next() {
		pods, missingIndices, err := ds.podStore.GetPodsWithIDs(podsToIndex[start:end]...)
		if err != nil {
			return err
		}
		if err := ds.podIndexer.AddPods(pods); err != nil {
			return err
		}
		if len(missingIndices) > 0 {
			idsToRemove := make([]string, 0, len(missingIndices))
			for _, missingIdx := range missingIndices {
				idsToRemove = append(idsToRemove, podsToIndex[start:end][missingIdx])
			}
			if err := ds.podIndexer.DeletePods(idsToRemove); err != nil {
				return err
			}
		}

		// Ack keys so that even if central restarts, we don't need to reindex them again
		if err := ds.podStore.AckKeysIndexed(podsToIndex[start:end]...); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d pods", end, len(podsToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync pods")
	return nil
}

func (ds *datastoreImpl) fullReindex() error {
	log.Info("[STARTUP] Reindexing all pods")

	podIDs, err := ds.podStore.GetPodIDs()
	if err != nil {
		return err
	}
	log.Infof("[STARTUP] Found %d pods to index", len(podIDs))
	podBatcher := batcher.New(len(podIDs), podBatchSize)
	for start, end, valid := podBatcher.Next(); valid; start, end, valid = podBatcher.Next() {
		pods, _, err := ds.podStore.GetPodsWithIDs(podIDs[start:end]...)
		if err != nil {
			return err
		}
		if err := ds.podIndexer.AddPods(pods); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d pods", end, len(podIDs))
	}
	log.Infof("[STARTUP] Successfully indexed %d pods", len(podIDs))

	// Clear the keys because we just re-indexed everything
	keys, err := ds.podStore.GetKeysToIndex()
	if err != nil {
		return err
	}
	if err := ds.podStore.AckKeysIndexed(keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.podIndexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.podSearcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Pod", "SearchRawPods")

	return ds.podSearcher.SearchRawPods(ctx, q)
}

func (ds *datastoreImpl) GetPod(ctx context.Context, id string) (*storage.Pod, bool, error) {
	pod, found, err := ds.podStore.GetPod(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := podsSAC.ReadAllowed(ctx, sac.KeyForNSScopedObj(pod)...); err != nil || !ok {
		return nil, false, err
	}
	return pod, true, nil
}

func (ds *datastoreImpl) GetPods(ctx context.Context, ids []string) ([]*storage.Pod, error) {
	if ok, err := podsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrPermissionDenied
	}

	pods, _, err := ds.podStore.GetPodsWithIDs(ids...)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (ds *datastoreImpl) CountPods(ctx context.Context) (int, error) {
	if ok, err := podsSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, sac.ErrPermissionDenied
	}

	return ds.podStore.CountPods()
}

// UpsertPod inserts a pod into podStore
func (ds *datastoreImpl) UpsertPod(ctx context.Context, pod *storage.Pod) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Pod", "UpsertPod")

	if ok, err := podsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	ds.processFilter.UpdateByPod(pod)

	err := ds.keyedMutex.DoStatusWithLock(pod.GetId(), func() error {
		if err := ds.podStore.UpsertPod(pod); err != nil {
			return errors.Wrapf(err, "inserting pod %q to store", pod.GetName())
		}
		if err := ds.podIndexer.AddPod(pod); err != nil {
			return errors.Wrapf(err, "inserting pod %q to index", pod.GetName())
		}
		if err := ds.podStore.AckKeysIndexed(pod.GetId()); err != nil {
			return errors.Wrapf(err, "could not acknowledge indexing for %q", pod.GetName())
		}
		return nil
	})
	if err != nil {
		return err
	}

	// For benchmark testing only. Ideally this is not nil in production.
	if ds.indicators == nil {
		return nil
	}

	deleteIndicatorsCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))
	return ds.indicators.RemoveProcessIndicatorsOfStaleContainersByPod(deleteIndicatorsCtx, pod)
}

// RemovePod removes a pod from the podStore
func (ds *datastoreImpl) RemovePod(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Pod", "RemovePod")

	if ok, err := podsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	pod, found, err := ds.podStore.GetPod(id)
	if err != nil || !found {
		return err
	}
	ds.processFilter.DeleteByPod(pod)

	err = ds.keyedMutex.DoStatusWithLock(id, func() error {
		if err := ds.podStore.RemovePod(id); err != nil {
			return err
		}
		if err := ds.podIndexer.DeletePod(id); err != nil {
			return err
		}
		if err := ds.podStore.AckKeysIndexed(id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	deleteIndicatorsCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))
	return ds.indicators.RemoveProcessIndicatorsByPod(deleteIndicatorsCtx, id)
}
