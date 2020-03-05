package datastore

import (
	"context"
	"fmt"
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
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/debug"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
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

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	return ds.searcher.SearchRawProcessIndicators(ctx, q)
}

func (ds *datastoreImpl) GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error) {
	indicator, exists, err := ds.storage.GetProcessIndicator(id)
	if err != nil || !exists {
		return nil, false, err
	}

	if ok, err := indicatorSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(indicator).Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return indicator, true, nil
}

func (ds *datastoreImpl) AddProcessIndicators(ctx context.Context, indicators ...*storage.ProcessIndicator) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	idSet := set.NewStringSet()
	var filteredIndicators []*storage.ProcessIndicator
	// Iterate over the processes in reverse chronological ordering
	// so we will maintain the most recent ones
	for i := len(indicators) - 1; i >= 0; i-- {
		if idSet.Add(indicators[i].GetId()) {
			filteredIndicators = append(filteredIndicators, indicators[i])
		}
	}

	err := ds.storage.AddProcessIndicators(filteredIndicators...)
	if err != nil {
		return err
	}

	if err := ds.indexer.AddProcessIndicators(filteredIndicators); err != nil {
		return err
	}

	filteredKeys := make([]string, 0, len(filteredIndicators))
	for _, fi := range filteredIndicators {
		filteredKeys = append(filteredKeys, fi.GetId())
	}
	if err := ds.storage.AckKeysIndexed(filteredKeys...); err != nil {
		return errors.Wrap(err, "error acknowledging added process indexing")
	}
	return nil
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error {
	if ok, err := indicatorSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.WalkAll(fn)
}

func (ds *datastoreImpl) RemoveProcessIndicators(ctx context.Context, ids []string) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.removeIndicators(ids)
}

func (ds *datastoreImpl) removeMatchingIndicators(results []pkgSearch.Result) error {
	idsToDelete := make([]string, 0, len(results))
	for _, r := range results {
		idsToDelete = append(idsToDelete, r.ID)
	}
	return ds.removeIndicators(idsToDelete)
}

func (ds *datastoreImpl) removeIndicators(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := ds.storage.RemoveProcessIndicators(ids); err != nil {
		return err
	}
	if err := ds.indexer.DeleteProcessIndicators(ids); err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(ids...); err != nil {
		return errors.Wrap(err, "error acknowledging indicator removal")
	}
	return nil
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByDeployment(ctx context.Context, id string) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, id).ProtoQuery()
	results, err := ds.Search(ctx, q)
	if err != nil {
		return err
	}
	return ds.removeMatchingIndicators(results)
}

func (ds *datastoreImpl) RemoveProcessIndicatorsOfStaleContainers(ctx context.Context, deployment *storage.Deployment) error {
	if ok, err := indicatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	mustConjunction := &v1.ConjunctionQuery{
		Queries: []*v1.Query{
			pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, deployment.GetId()).ProtoQuery(),
			pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.DeploymentStateTS, fmt.Sprintf("<=%d", deployment.GetStateTimestamp())).ProtoQuery(),
		},
	}

	currentContainerIDs := containerIds(deployment)
	queries := make([]*v1.Query, 0, len(currentContainerIDs))
	for _, containerID := range currentContainerIDs {
		queries = append(queries, pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ContainerID, pkgSearch.ExactMatchString(containerID)).ProtoQuery())
	}

	mustNotDisjunction := &v1.DisjunctionQuery{
		Queries: queries,
	}

	booleanQuery := pkgSearch.NewBooleanQuery(mustConjunction, mustNotDisjunction)

	results, err := ds.Search(ctx, booleanQuery)
	if err != nil {
		return err
	}
	return ds.removeMatchingIndicators(results)
}

func (ds *datastoreImpl) prunePeriodically() {
	defer ds.stoppedSig.Signal()

	if ds.prunerFactory == nil {
		return
	}

	t := time.NewTicker(ds.prunerFactory.Period())
	defer t.Stop()
	for !ds.stopSig.IsDone() {
		select {
		case <-t.C:
			ds.prune()
		case <-ds.stopSig.Done():
			return
		}
	}
}

func (ds *datastoreImpl) prune() {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Prune, "ProcessIndicator")
	pruner := ds.prunerFactory.StartPruning()
	defer pruner.Finish()

	processInfoToArgs, err := ds.storage.GetProcessInfoToArgs()
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
			if err := ds.removeIndicators(idsToRemove); err != nil {
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

func (ds *datastoreImpl) fullReindex() error {
	log.Info("[STARTUP] Reindexing all processes")

	indicators := make([]*storage.ProcessIndicator, 0, maxBatchSize)
	var count int
	err := ds.storage.WalkAll(func(pi *storage.ProcessIndicator) error {
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
	keys, err := ds.storage.GetKeysToIndex()
	if err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.indexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()

	needsFullIndexing, err := ds.indexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsFullIndexing {
		return ds.fullReindex()
	}
	log.Info("[STARTUP] Determining if process db/indexer reconciliation is needed")
	processesToIndex, err := ds.storage.GetKeysToIndex()
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index")
	}

	log.Infof("[STARTUP] Found %d Processes to index", len(processesToIndex))

	processBatcher := batcher.New(len(processesToIndex), maxBatchSize)
	for start, end, valid := processBatcher.Next(); valid; start, end, valid = processBatcher.Next() {
		processes, missingIndices, err := ds.storage.GetBatchProcessIndicators(processesToIndex[start:end])
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
		if err := ds.storage.AckKeysIndexed(processesToIndex[start:end]...); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d processes", end, len(processesToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync processes")
	return nil
}

func containerIds(deployment *storage.Deployment) (ids []string) {
	for _, container := range deployment.GetContainers() {
		for _, instance := range container.GetInstances() {
			containerID := containerid.ShortContainerIDFromInstance(instance)
			if containerID != "" {
				ids = append(ids, containerID)
			}
		}
	}
	return
}
