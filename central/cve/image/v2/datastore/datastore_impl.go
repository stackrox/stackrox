package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/search"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchImageCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error) {
	cves, err := ds.searcher.SearchRawImageCVEs(ctx, q)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEV2, bool, error) {
	cve, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageCVEV2, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()

	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCve()]; ok {
				vuln.Suppressed = true
				vuln.SuppressActivation = protocompat.ConvertTimeToTimestampOrNil(entry.SuppressActivation)
				vuln.SuppressExpiry = protocompat.ConvertTimeToTimestampOrNil(entry.SuppressExpiry)
				vuln.State = storage.VulnerabilityState_DEFERRED
			}
		}
	}
}
