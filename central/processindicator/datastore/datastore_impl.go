package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
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
	maxBatchSize = 5000
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

	// Clean up correlated ProcessListeningOnPort objects. Probably could be
	// done using a proper FK and CASCADE, but it's usually a thing you would
	// not like to do automatically. search.ProcessID is not a PID, but UUID of
	// the record in the table

	// Batch the deletes as scaled pruning can result in many process indicators being deleted.
	localBatchSize := maxBatchSize
	numRecordsToDelete := len(ids)
	for {
		if len(ids) == 0 {
			break
		}

		if len(ids) < localBatchSize {
			localBatchSize = len(ids)
		}

		identifierBatch := ids[:localBatchSize]
		deleteQuery := pkgSearch.NewQueryBuilder().
			AddStrings(pkgSearch.ProcessID, identifierBatch...).
			ProtoQuery()

		if err := ds.plopStorage.DeleteByQuery(ctx, deleteQuery); err != nil {
			err = errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(ids), numRecordsToDelete)
			return err
		}

		// Move the slice forward to start the next batch
		ids = ids[localBatchSize:]
	}

	return nil
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByPod(ctx context.Context, id string) error {
	if ok, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PodUID, id).ProtoQuery()
	return ds.storage.DeleteByQuery(ctx, q)
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

func (ds *datastoreImpl) Stop() {
	ds.stopper.Client().Stop()
}

func (ds *datastoreImpl) Wait(cancelWhen concurrency.Waitable) bool {
	return concurrency.WaitInContext(ds.stopper.Client().Stopped(), cancelWhen)
}
