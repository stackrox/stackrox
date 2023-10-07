package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	podSearch "github.com/stackrox/rox/central/pod/datastore/internal/search"
	podStore "github.com/stackrox/rox/central/pod/store"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	plopDS "github.com/stackrox/rox/central/processlisteningonport/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	resourceType              = "Pod"
	maxNumberOfDeadContainers = 10
)

var (
	// It should not be possible that pod and deployment scope are different,
	// so just use the same access controls as a deployment.
	podsSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	podStore    podStore.Store
	podSearcher podSearch.Searcher

	indicators    piDS.DataStore
	plops         plopDS.DataStore
	processFilter filter.Filter

	keyedMutex *concurrency.KeyedMutex
}

func newDatastoreImpl(storage podStore.Store, searcher podSearch.Searcher, indicators piDS.DataStore, plops plopDS.DataStore, processFilter filter.Filter) *datastoreImpl {
	return &datastoreImpl{
		podStore:      storage,
		podSearcher:   searcher,
		indicators:    indicators,
		plops:         plops,
		processFilter: processFilter,
		keyedMutex:    concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.podSearcher.Search(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.podSearcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "SearchRawPods")

	return ds.podSearcher.SearchRawPods(ctx, q)
}

func (ds *datastoreImpl) GetPod(ctx context.Context, id string) (*storage.Pod, bool, error) {
	pod, found, err := ds.podStore.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := podsSAC.ReadAllowed(ctx, sac.KeyForNSScopedObj(pod)...); err != nil || !ok {
		return nil, false, err
	}
	return pod, true, nil
}

// UpsertPod inserts a pod into podStore
func (ds *datastoreImpl) UpsertPod(ctx context.Context, pod *storage.Pod) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Upsert")

	if ok, err := podsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.processFilter.UpdateByPod(pod)

	err := ds.keyedMutex.DoStatusWithLock(pod.GetId(), func() error {
		oldPod, found, err := ds.podStore.Get(ctx, pod.GetId())
		if err != nil {
			return errors.Wrapf(err, "retrieving pod %q from store", pod.GetName())
		}
		if found {
			mergeContainerInstances(pod, oldPod)
		}

		if err := ds.podStore.Upsert(ctx, pod); err != nil {
			return errors.Wrapf(err, "inserting pod %q to store", pod.GetName())
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// mergeContainerInstances merges container instances from oldPod into newPod.
func mergeContainerInstances(newPod *storage.Pod, oldPod *storage.Pod) {
	newPod.TerminatedInstances = oldPod.TerminatedInstances

	idxByContainerName := make(map[string]int)
	for i, instanceList := range newPod.GetTerminatedInstances() {
		if len(instanceList.GetInstances()) > 0 {
			idxByContainerName[instanceList.GetInstances()[0].GetContainerName()] = i
		}
	}

	endIdx := 0
	for _, instance := range newPod.GetLiveInstances() {
		if instance.GetFinished() == nil {
			newPod.LiveInstances[endIdx] = instance
			endIdx++
		} else {
			// Container Instance has terminated. Move it into the proper dead instances list.
			if idx, exists := idxByContainerName[instance.GetContainerName()]; exists {
				deadInstancesList := newPod.GetTerminatedInstances()[idx]
				var startIdx int
				if len(deadInstancesList.Instances) == maxNumberOfDeadContainers {
					// Remove the oldest entry.
					startIdx = 1
				}
				deadInstancesList.Instances = append(deadInstancesList.Instances[startIdx:], instance)
			} else {
				newPod.TerminatedInstances = append(newPod.TerminatedInstances, &storage.Pod_ContainerInstanceList{
					Instances: []*storage.ContainerInstance{instance},
				})
			}
		}
	}
	newPod.LiveInstances = newPod.LiveInstances[:endIdx]
}

// RemovePod removes a pod from the podStore
func (ds *datastoreImpl) RemovePod(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Delete")

	if ok, err := podsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	pod, found, err := ds.podStore.Get(ctx, id)
	if err != nil || !found {
		return err
	}
	ds.processFilter.DeleteByPod(pod)

	err = ds.keyedMutex.DoStatusWithLock(id, func() error {
		return ds.podStore.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	deleteIndicatorsCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))

	errIndicators := ds.indicators.RemoveProcessIndicatorsByPod(deleteIndicatorsCtx, id)

	errPlop := ds.plops.RemovePlopsByPod(deleteIndicatorsCtx, id)

       if errInidicators != nil {
               return errIndicators
       }

       return errPlop
}

func (ds *datastoreImpl) GetPodIDs(ctx context.Context) ([]string, error) {
	return ds.podStore.GetIDs(ctx)
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(pod *storage.Pod) error) error {
	if ok, err := podsSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	// Postgres retry in caller.
	return ds.podStore.Walk(ctx, fn)
}
