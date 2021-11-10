package datastore

import (
	"context"

	sacFilters "github.com/stackrox/rox/central/imagecveedge/sac"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type datastoreImpl struct {
	graphProvider graph.Provider
	storage       store.Store
	searcher      search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	if !features.VulnRiskManagement.Enabled() {
		return nil, status.Error(codes.FailedPrecondition, "Vulnerability Risk Management is not enabled")
	}
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if !features.VulnRiskManagement.Enabled() {
		return nil, status.Error(codes.FailedPrecondition, "Vulnerability Risk Management is not enabled")
	}
	return ds.searcher.SearchEdges(ctx, q)
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
	if !features.VulnRiskManagement.Enabled() {
		return nil, status.Error(codes.FailedPrecondition, "Vulnerability Risk Management is not enabled")
	}
	edges, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	edge, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
