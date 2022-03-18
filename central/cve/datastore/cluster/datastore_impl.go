package cluster

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/datastore/cluster/internal/store/postgres"
	"github.com/stackrox/rox/central/cve/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage  postgres.Store
	searcher search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	return ds.searcher.SearchRawCVEs(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.CVE, bool, error) {
	cve, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	return ds.storage.Exists(ctx, id)
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Suppress(_ context.Context, _ *types.Timestamp, _ *types.Duration, _ ...string) error {
	return errors.New("vulnerability snoozing/un-snoozing is not supported cluster (k8s/istio) vulnerabilities")
}

func (ds *datastoreImpl) Unsuppress(ctx context.Context, ids ...string) error {
	return errors.New("vulnerability snoozing/un-snoozing is not supported cluster (k8s/istio) vulnerabilities")
}

func (ds *datastoreImpl) EnrichImageWithSuppressedCVEs(_ *storage.Image) {}

func (ds *datastoreImpl) EnrichNodeWithSuppressedCVEs(_ *storage.Node) {}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	return ds.storage.DeleteMany(ctx, ids)
}
