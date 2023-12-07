package datastore

import (
	"context"

	"github.com/stackrox/rox/central/policycategoryedge/search"
	"github.com/stackrox/rox/central/policycategoryedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	workflowAdministrationSAC = sac.ForResource(resources.WorkflowAdministration)
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	mutex sync.Mutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchEdges(ctx, q)
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategoryEdge, error) {
	imgs, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.PolicyCategoryEdge, bool, error) {
	edge, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

// GetAllPolicyCategories lists all policy categories
func (ds *datastoreImpl) GetAll(ctx context.Context) ([]*storage.PolicyCategoryEdge, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var edges []*storage.PolicyCategoryEdge
	err := ds.storage.Walk(ctx, func(edge *storage.PolicyCategoryEdge) error {
		edges = append(edges, edge)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return edges, err
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.PolicyCategoryEdge, error) {
	edges, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) UpsertMany(ctx context.Context, edges []*storage.PolicyCategoryEdge) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if len(edges) == 0 {
		return nil
	}

	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// Store the new category association data.
	return ds.storage.UpsertMany(ctx, edges)
}

func (ds *datastoreImpl) DeleteMany(ctx context.Context, ids ...string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) DeleteByQuery(ctx context.Context, q *v1.Query) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	_, storeErr := ds.storage.DeleteByQuery(ctx, q)
	return storeErr
}
