package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/features"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	maxBatchSize = 5000
)

var (
	indicatorSAC = sac.ForResource(resources.Indicator)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	prunerFactory         pruner.Factory
	prunedArgsLengthCache map[processindicator.ProcessWithContainerInfo]int

	stopSig, stoppedSig concurrency.Signal
}

func checkReadAccess(ctx context.Context, indicator *storage.ProcessIndicator) (bool, error) {
	return indicatorSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(indicator).Allowed(ctx)
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
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	err := ds.storage.UpsertMany(ctx, indicators)
	if err != nil {
		return err
	}

	if err := ds.indexer.AddProcessIndicators(indicators); err != nil {
		return err
	}

	keys := make([]string, 0, len(indicators))
	for _, indicator := range indicators {
		keys = append(keys, indicator.GetId())
	}
	if err := ds.storage.AckKeysIndexed(ctx, keys...); err != nil {
		return errors.Wrap(err, "error acknowledging added process indexing")
	}
	return nil
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error {
	if ok, err := indicatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Walk(ctx, fn)
}

func (ds *datastoreImpl) RemoveProcessIndicators(ctx context.Context, ids []string) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.removeIndicators(ctx, ids)
}

func (ds *datastoreImpl) removeMatchingIndicators(ctx context.Context, results []pkgSearch.Result) error {
	return ds.removeIndicators(ctx, pkgSearch.ResultsToIDs(results))
}

func (ds *datastoreImpl) removeIndicators(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return err
	}
	if err := ds.indexer.DeleteProcessIndicators(ids); err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(ctx, ids...); err != nil {
		return errors.Wrap(err, "error acknowledging indicator removal")
	}
	return nil
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByPod(ctx context.Context, id string) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PodUID, id).ProtoQuery()
	results, err := ds.Search(ctx, q)
	if err != nil {
		return err
	}
	return ds.removeMatchingIndicators(ctx, results)
}

func (ds *datastoreImpl) prunePeriodically(ctx context.Context) {
	defer ds.stoppedSig.Signal()

	if ds.prunerFactory == nil {
		return
	}

	t := time.NewTicker(ds.prunerFactory.Period())
	defer t.Stop()
	for !ds.stopSig.IsDone() {
		select {
		case <-t.C:
			ds.prune(ctx)
		case <-ds.stopSig.Done():
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
		if len(idsToRemove) > 0 {
			if err := ds.removeIndicators(ctx, idsToRemove); err != nil {
				log.Errorf("Error while pruning processes: %s", err)
			} else {
				incrementPrunedProcessesMetric(len(idsToRemove))
			}
		}
		ds.prunedArgsLengthCache[processInfo] = numArgsReceived - len(idsToRemove)
	}

	// Clean up the prunedArgsLengthCache by processes that are no longer in the DB.
	for processInfo := range ds.prunedArgsLengthCache {
		if _, exists := processInfoToArgs[processInfo]; !exists {
			delete(ds.prunedArgsLengthCache, processInfo)
		}
	}
}

func (ds *datastoreImpl) Stop() bool {
	return ds.stopSig.Signal()
}

func (ds *datastoreImpl) Wait(cancelWhen concurrency.Waitable) bool {
	return concurrency.WaitInContext(&ds.stoppedSig, cancelWhen)
}

func (ds *datastoreImpl) fullReindex(ctx context.Context) error {
	log.Info("[STARTUP] Reindexing all processes")

	indicators := make([]*storage.ProcessIndicator, 0, maxBatchSize)
	var count int
	err := ds.storage.Walk(ctx, func(pi *storage.ProcessIndicator) error {
		indicators = append(indicators, pi)
		if len(indicators) == maxBatchSize {
			if err := ds.indexer.AddProcessIndicators(indicators); err != nil {
				return err
			}
			count += maxBatchSize
			indicators = indicators[:0]
			log.Infof("[STARTUP] Successfully indexed %d processes", count)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := ds.indexer.AddProcessIndicators(indicators); err != nil {
		return err
	}
	count += len(indicators)
	log.Infof("[STARTUP] Successfully indexed all %d processes", count)

	// Clear the keys because we just re-indexed everything
	keys, err := ds.storage.GetKeysToIndex(ctx)
	if err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(ctx, keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.indexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()

	needsFullIndexing, err := ds.indexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsFullIndexing {
		return ds.fullReindex(ctx)
	}
	log.Info("[STARTUP] Determining if process db/indexer reconciliation is needed")
	processesToIndex, err := ds.storage.GetKeysToIndex(ctx)
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index")
	}

	log.Infof("[STARTUP] Found %d Processes to index", len(processesToIndex))

	processBatcher := batcher.New(len(processesToIndex), maxBatchSize)
	for start, end, valid := processBatcher.Next(); valid; start, end, valid = processBatcher.Next() {
		processes, missingIndices, err := ds.storage.GetMany(ctx, processesToIndex[start:end])
		if err != nil {
			return err
		}
		if err := ds.indexer.AddProcessIndicators(processes); err != nil {
			return err
		}
		if len(missingIndices) > 0 {
			idsToRemove := make([]string, 0, len(missingIndices))
			for _, missingIdx := range missingIndices {
				idsToRemove = append(idsToRemove, processesToIndex[start:end][missingIdx])
			}
			if err := ds.indexer.DeleteProcessIndicators(idsToRemove); err != nil {
				return err
			}
		}

		// Ack keys so that even if central restarts, we don't need to reindex them again
		if err := ds.storage.AckKeysIndexed(ctx, processesToIndex[start:end]...); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d processes", end, len(processesToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync processes")
	return nil
}
