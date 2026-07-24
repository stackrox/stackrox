package datastore

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	aiWorkloadStore "github.com/stackrox/rox/central/aiworkload/datastore/internal/store"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	defaultPageSize = 100
	preAllocateCap  = math.MaxUint16
)

type datastoreImpl struct {
	store aiWorkloadStore.AIWorkloadStore
	mutex sync.Mutex
}

func newDatastoreImpl(store aiWorkloadStore.AIWorkloadStore) DataStore {
	return &datastoreImpl{store: store}
}

func (ds *datastoreImpl) CountAIWorkloads(ctx context.Context, query *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "CountAIWorkloads")
	if query == nil {
		query = search.EmptyQuery()
	}
	return ds.store.Count(ctx, query)
}

func (ds *datastoreImpl) GetAIWorkload(ctx context.Context, id string) (*storage.AIWorkload, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "GetAIWorkload")
	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) UpsertAIWorkload(ctx context.Context, workload *storage.AIWorkload) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "UpsertAIWorkload")

	if workload.GetId() == "" {
		return errors.New("cannot upsert an AI workload without an id")
	}

	now := time.Now()
	workload.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&now)

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	return ds.store.UpsertMany(ctx, []*storage.AIWorkload{workload})
}

func (ds *datastoreImpl) DeleteAIWorkloads(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "DeleteAIWorkloads")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	return ds.store.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "Exists")
	return ds.store.Exists(ctx, id)
}

func (ds *datastoreImpl) SearchRawAIWorkloads(ctx context.Context, query *v1.Query) ([]*storage.AIWorkload, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "SearchRawAIWorkloads")

	if query == nil {
		query = search.EmptyQuery()
	}
	searchQuery := query.CloneVT()
	if len(searchQuery.GetPagination().GetSortOptions()) == 0 {
		if searchQuery.GetPagination() == nil {
			searchQuery.Pagination = &v1.QueryPagination{}
		}
		searchQuery.Pagination.SortOptions = []*v1.QuerySortOption{
			{Field: search.AIWorkloadName.String()},
			{Field: search.Namespace.String()},
		}
	}
	pageSize := searchQuery.GetPagination().GetLimit()
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > preAllocateCap {
		pageSize = preAllocateCap
	}
	results := make([]*storage.AIWorkload, 0, pageSize)
	err := ds.store.WalkByQuery(ctx, searchQuery, func(w *storage.AIWorkload) error {
		results = append(results, w)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (ds *datastoreImpl) Walk(ctx context.Context, fn func(w *storage.AIWorkload) error) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "AIWorkload", "Walk")
	return ds.store.Walk(ctx, fn)
}
