package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

type datastoreImpl struct {
	storage store.Store

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	cves, missingIndices, err := ds.storage.GetMany(ctx, pkgSearch.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	results = pkgSearch.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results)
}

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error) {
	var cves []*storage.ImageCVEV2
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.ImageCVEV2) error {
		cves = append(cves, cve)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cves, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
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
				vuln.SetSuppressed(true)
				vuln.SetSuppressActivation(protocompat.ConvertTimeToTimestampOrNil(entry.SuppressActivation))
				vuln.SetSuppressExpiry(protocompat.ConvertTimeToTimestampOrNil(entry.SuppressExpiry))
				vuln.SetState(storage.VulnerabilityState_DEFERRED)
			}
		}
	}
}

func convertMany(cves []*storage.ImageCVEV2, results []pkgSearch.Result) ([]*v1.SearchResult, error) {
	if len(cves) != len(results) {
		return nil, errors.Errorf("expected %d CVEs but got %d", len(results), len(cves))
	}

	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(cve *storage.ImageCVEV2, result *pkgSearch.Result) *v1.SearchResult {
	sr := &v1.SearchResult{}
	sr.SetCategory(v1.SearchCategory_IMAGE_VULNERABILITIES_V2)
	sr.SetId(cve.GetId())
	sr.SetName(cve.GetCveBaseInfo().GetCve())
	sr.SetFieldToMatches(pkgSearch.GetProtoMatchesMap(result.Matches))
	sr.SetScore(result.Score)
	return sr
}
