package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	deleteBatchSize = 5000
)

var (
	deploymentExtensionSAC = sac.ForResource(resources.DeploymentExtension)
)

type datastoreImpl struct {
	storage store.Store
	// ProcessListeningOnPort storage is needed for correct pruning. It
	// logically belongs to the datastore implementation of PLOP, but this way
	// it would be an import cycle, so call the Store directly.
	plopStorage plopStore.Store
	searcher    search.Searcher

	prunerFactory         pruner.Factory
	prunedArgsLengthCache map[processindicator.ProcessWithContainerInfo]int

	stopper concurrency.Stopper
}

func checkReadAccess(ctx context.Context, indicator *storage.ProcessIndicator) (bool, error) {
	return deploymentExtensionSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(indicator).IsAllowed(), nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	return ds.searcher.SearchRawProcessIndicators(ctx, q)
}

func (ds *datastoreImpl) GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error) {
	indicator, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	if ok, err := checkReadAccess(ctx, indicator); !ok || err != nil {
		return nil, false, err
	}

	return indicator, true, nil
}

func (ds *datastoreImpl) GetProcessIndicators(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, bool, error) {
	indicators, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil || len(indicators) == 0 {
		return nil, false, err
	}

	allowedIndicators := indicators[:0]

	for _, indicator := range indicators {
		if ok, err := checkReadAccess(ctx, indicator); !ok || err != nil {
			continue
		}

		allowedIndicators = append(allowedIndicators, indicator)
	}

	return allowedIndicators, len(allowedIndicators) != 0, nil
}

func (ds *datastoreImpl) AddProcessIndicators(ctx context.Context, indicators ...*storage.ProcessIndicator) error {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.UpsertMany(ctx, indicators)
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error {
	if ok, err := deploymentExtensionSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Walk(ctx, fn)
}

func (ds *datastoreImpl) RemoveProcessIndicators(ctx context.Context, ids []string) error {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.removeIndicators(ctx, ids)
}

func (ds *datastoreImpl) removeIndicators(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) PruneProcessIndicators(ctx context.Context, ids []string) (int, error) {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, sac.ErrResourceAccessDenied
	}

	return ds.pruneIndicators(ctx, ids), nil
}

func (ds *datastoreImpl) pruneIndicators(ctx context.Context, ids []string) int {
	// Previously this used removeIndicators and would call "DeleteMany".  The issue
	// with that is "DeleteMany" wraps the entire delete into a transaction making it an
	// all or nothing proposition.  For pruning, if a batch fails it shouldn't fail them all.
	// A pruning batch that fails to delete would get deleted the next iteration of pruning.
	// So for pruning, a delete by query will be used and the IDs will be batched.  Failed
	// batches will be logged and we will move on to the next batch.
	if len(ids) == 0 {
		return 0
	}

	// Batch the deletes
	initialSize := len(ids)
	localBatchSize := deleteBatchSize
	var successfullyPruned int
	for {
		if len(ids) == 0 {
			break
		}

		if len(ids) < localBatchSize {
			localBatchSize = len(ids)
		}

		identifierBatch := ids[:localBatchSize]

		q := pkgSearch.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery()

		deletedIDs, err := ds.storage.DeleteByQuery(ctx, q)
		if err != nil {
			log.Warnf("error pruning a batch of indicators: %v", err)
		} else {
			successfullyPruned = successfullyPruned + len(deletedIDs)
			log.Debugf("successfully pruned a batch of %d process indicators", len(deletedIDs))
		}

		// Move the slice forward to start the next batch
		ids = ids[localBatchSize:]
	}

	log.Infof("successfully pruned %d out of %d indicators", successfullyPruned, initialSize)
	return successfullyPruned
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByPod(ctx context.Context, id string) error {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PodUID, id).ProtoQuery()
	_, storeErr := ds.storage.DeleteByQuery(ctx, q)
	return storeErr
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByPods(ctx context.Context, ids []string) error {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PodUID, ids...).ProtoQuery()
	_, storeErr := ds.storage.DeleteByQuery(ctx, q)
	return storeErr
}

func (ds *datastoreImpl) prunePeriodically(ctx context.Context) {
	defer ds.stopper.Flow().ReportStopped()

	if ds.prunerFactory == nil {
		return
	}

	t := time.NewTicker(ds.prunerFactory.Period())
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ds.prune(ctx)
		case <-ds.stopper.Flow().StopRequested():
			return
		}
	}
}

func (ds *datastoreImpl) getProcessInfoToArgs(ctx context.Context) (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ProcessIndicator", "getProcessInfoToArgs")
	processNamesToArgs := make(map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs)
	err := ds.storage.Walk(ctx, func(pi *storage.ProcessIndicator) error {
		info := processindicator.ProcessWithContainerInfo{
			ContainerName: pi.GetContainerName(),
			PodID:         pi.GetPodId(),
			ProcessName:   pi.GetSignal().GetName(),
		}
		processNamesToArgs[info] = append(processNamesToArgs[info], processindicator.IDAndArgs{
			ID:   pi.GetId(),
			Args: pi.GetSignal().GetArgs(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return processNamesToArgs, nil
}

func (ds *datastoreImpl) prune(ctx context.Context) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Prune, "ProcessIndicator")
	pruner := ds.prunerFactory.StartPruning()
	defer pruner.Finish()

	processInfoToArgs, err := ds.getProcessInfoToArgs(ctx)
	if err != nil {
		log.Errorf("Error while pruning processes: couldn't retrieve process info to args: %s", err)
		return
	}

	for processInfo, args := range processInfoToArgs {
		numArgsReceived := len(args)
		if previouslyPrunedArgsLength, found := ds.prunedArgsLengthCache[processInfo]; found {
			if previouslyPrunedArgsLength == numArgsReceived {
				incrementProcessPruningCacheHitsMetrics()
				continue
			}
		}
		incrementProcessPruningCacheMissesMetric()
		idsToRemove := pruner.Prune(args)
		var successfullyPruned int
		if len(idsToRemove) > 0 {
			successfullyPruned = ds.pruneIndicators(ctx, idsToRemove)
			incrementPrunedProcessesMetric(successfullyPruned)
		}
		ds.prunedArgsLengthCache[processInfo] = numArgsReceived - successfullyPruned
	}

	// Clean up the prunedArgsLengthCache by processes that are no longer in the DB.
	for processInfo := range ds.prunedArgsLengthCache {
		if _, exists := processInfoToArgs[processInfo]; !exists {
			delete(ds.prunedArgsLengthCache, processInfo)
		}
	}
}

func (ds *datastoreImpl) Stop() {
	ds.stopper.Client().Stop()
}

func (ds *datastoreImpl) Wait(cancelWhen concurrency.Waitable) bool {
	return concurrency.WaitInContext(ds.stopper.Client().Stopped(), cancelWhen)
}
