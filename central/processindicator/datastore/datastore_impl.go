package datastore

import (
	"context"
	"errors"
	"slices"

	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/processindicator/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

const (
	deleteBatchSize = 5000
	getBatchSize    = 1000
)

var (
	addBatchSize = env.ProcessAddBatchSize.IntegerSetting()
)

type datastoreImpl struct {
	db postgres.DB

	storage store.Store

	stopper concurrency.Stopper
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	return ds.storage.GetByQuery(ctx, q)
}

func (ds *datastoreImpl) GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.ProcessIndicator) error) error {
	return ds.storage.GetByQueryFn(ctx, query, fn)
}

func (ds *datastoreImpl) GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error) {
	indicator, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	return indicator, true, nil
}

func (ds *datastoreImpl) GetProcessIndicators(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, bool, error) {
	indicators := make([]*storage.ProcessIndicator, 0, len(ids))

	for idsBatch := range slices.Chunk(ids, getBatchSize) {
		batchIndicators, _, err := ds.storage.GetMany(ctx, idsBatch)

		if err != nil {
			return nil, false, err
		}

		indicators = append(indicators, batchIndicators...)
	}

	if len(indicators) == 0 {
		return nil, false, nil
	}

	return indicators, len(indicators) != 0, nil
}

func (ds *datastoreImpl) AddProcessIndicators(ctx context.Context, indicators ...*storage.ProcessIndicator) error {
	for identifierBatch := range slices.Chunk(indicators, addBatchSize) {
		err := ds.storage.UpsertMany(ctx, identifierBatch)
		if err != nil {
			log.Warnf("error adding a batch of indicators: %v", err)
			if errors.Is(err, sac.ErrResourceAccessDenied) {
				return err
			}
		} else {
			recordProcessIndicatorsBatchAdded(identifierBatch)
			log.Debugf("successfully added a batch of %d process indicators", len(identifierBatch))
		}
	}

	return nil
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(pi *storage.ProcessIndicator) error) error {
	return ds.storage.WalkByQuery(ctx, q, fn)
}

func (ds *datastoreImpl) RemoveProcessIndicators(ctx context.Context, ids []string, reason string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return err
	}

	recordProcessIndicatorsRemoved(len(ids), reason)
	return nil
}

func (ds *datastoreImpl) PruneProcessIndicators(ctx context.Context, ids []string, reason string) (int, error) {
	return ds.pruneIndicators(ctx, ids, reason), nil
}

func (ds *datastoreImpl) pruneIndicators(ctx context.Context, ids []string, reason string) int {
	// Previously this used removeIndicators and would call "DeleteMany".  The issue
	// with that is "DeleteMany" wraps the entire delete into a transaction making it an
	// all or nothing proposition.  For pruning, if a batch fails it shouldn't fail them all.
	// A pruning batch that fails to delete would get deleted the next iteration of pruning.
	// So for pruning, a delete by query will be used and the IDs will be batched.  Failed
	// batches will be logged and we will move on to the next batch.
	if len(ids) == 0 {
		return 0
	}

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

		err := ds.storage.DeleteByQuery(ctx, q)
		if err != nil {
			log.Warnf("error pruning a batch of indicators: %v", err)
		} else {
			successfullyPruned = successfullyPruned + len(identifierBatch)
			log.Debugf("successfully pruned a batch of %d process indicators", len(identifierBatch))
		}

		ids = ids[localBatchSize:]
	}

	log.Infof("successfully pruned %d out of %d indicators", successfullyPruned, initialSize)
	recordProcessIndicatorsRemoved(successfullyPruned, reason)
	return successfullyPruned
}

func (ds *datastoreImpl) RemoveProcessIndicatorsByPod(ctx context.Context, id string) error {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.PodUID, id).ProtoQuery()

	deletedIDs, err := ds.storage.DeleteByQueryWithIDs(ctx, q)
	if err != nil {
		return err
	}

	if len(deletedIDs) > 0 {
		recordProcessIndicatorsRemoved(len(deletedIDs), RemovalReasonPodDeletion)
	}
	return nil
}

// IterateOverProcessIndicatorsRiskView iterates over minimal fields from process indicator for risk evaluation
func (ds *datastoreImpl) IterateOverProcessIndicatorsRiskView(ctx context.Context, q *v1.Query, fn func(*views.ProcessIndicatorRiskView) error) error {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.ProcessID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ContainerName).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ProcessExecPath).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ProcessContainerStartTime).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ProcessCreationTime).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ProcessName).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ProcessArguments).Proto(),
	}

	err := pgSearch.RunSelectRequestForSchemaFn[views.ProcessIndicatorRiskView](ctx, ds.db, pkgSchema.ProcessIndicatorsSchema, cloned, fn)
	if err != nil {
		log.Errorf("unable to iterate over indicators for risk processing: %v", err)
	}

	return err
}

func (ds *datastoreImpl) Stop() {
	ds.stopper.Client().Stop()
}

func (ds *datastoreImpl) Wait(cancelWhen concurrency.Waitable) bool {
	return concurrency.WaitInContext(ds.stopper.Client().Stopped(), cancelWhen)
}
