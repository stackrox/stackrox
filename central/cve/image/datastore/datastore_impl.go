package datastore

import (
	"context"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vulnRequesterOrApproverSAC = sac.ForResources(
		sac.ForResource(resources.VulnerabilityManagementRequests),
		sac.ForResource(resources.VulnerabilityManagementApprovals),
	)

	accessAllCtx = sac.WithAllAccess(context.Background())

	errNilSuppressionStart = errors.New("suppression start time is nil")

	errNilSuppressionDuration = errors.New("suppression duration is nil")
)

type datastoreImpl struct {
	storage store.Store

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache

	keyFence concurrency.KeyFence
}

func getSuppressionCacheEntry(cve *storage.ImageCVE) common.SuppressionCacheEntry {
	cacheEntry := common.SuppressionCacheEntry{}
	cacheEntry.SuppressActivation = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeStart())
	cacheEntry.SuppressExpiry = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeExpiry())
	return cacheEntry
}

func (ds *datastoreImpl) buildSuppressedCache() {
	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.SearchRawImageCVEs(accessAllCtx, query)
	if err != nil {
		log.Error(errors.Wrap(err, "Vulnerability exception management may not function correctly. Failed to build cache of CVE exceptions."))
		return
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range suppressedCVEs {
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(cve)
	}
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

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVE, error) {
	var cves []*storage.ImageCVE
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.ImageCVE) error {
		cves = append(cves, cve)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cves, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVE, bool, error) {
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageCVE, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Suppress(ctx context.Context, start *time.Time, duration *time.Duration, cves ...string) error {
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	expiry, err := getSuppressExpiry(start, duration)
	if err != nil {
		return err
	}

	vulns, err := ds.SearchRawImageCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	err = ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(gatherKeys(vulns)...), func() error {
		for _, vuln := range vulns {
			vuln.Snoozed = true
			vuln.SnoozeStart = protocompat.ConvertTimeToTimestampOrNil(start)
			vuln.SnoozeExpiry = protocompat.ConvertTimeToTimestampOrNil(expiry)
		}
		return ds.storage.UpsertMany(ctx, vulns)
	})
	if err != nil {
		return err
	}

	ds.updateCache(vulns...)
	return nil
}

func (ds *datastoreImpl) Unsuppress(ctx context.Context, cves ...string) error {
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	vulns, err := ds.SearchRawImageCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	err = ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(gatherKeys(vulns)...), func() error {
		for _, vuln := range vulns {
			vuln.Snoozed = false
			vuln.SnoozeStart = nil
			vuln.SnoozeExpiry = nil
		}
		return ds.storage.UpsertMany(ctx, vulns)
	})
	if err != nil {
		return err
	}

	ds.deleteFromCache(vulns...)
	return nil
}

func (ds *datastoreImpl) ApplyException(ctx context.Context, start, expiry *time.Time, cves ...string) error {
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	vulns, err := ds.SearchRawImageCVEs(ctx,
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	return ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(gatherKeys(vulns)...), func() error {
		for _, vuln := range vulns {
			vuln.Snoozed = true
			vuln.SnoozeStart = protocompat.ConvertTimeToTimestampOrNil(start)
			vuln.SnoozeExpiry = protocompat.ConvertTimeToTimestampOrNil(expiry)
		}
		return ds.storage.UpsertMany(ctx, vulns)
	})
}

func (ds *datastoreImpl) RevertException(ctx context.Context, cves ...string) error {
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	vulns, err := ds.SearchRawImageCVEs(ctx,
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	return ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(gatherKeys(vulns)...), func() error {
		for _, vuln := range vulns {
			vuln.Snoozed = false
			vuln.SnoozeStart = nil
			vuln.SnoozeExpiry = nil
		}
		return ds.storage.UpsertMany(ctx, vulns)
	})
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

func (ds *datastoreImpl) EnrichImageV2WithSuppressedCVEs(image *storage.ImageV2) {
	// do nothing, this is a no-op for the normalized CVE datastore.
	// Had to add this to satisfy the changes to the CVESuppressor interface.
}

func getSuppressExpiry(start *time.Time, duration *time.Duration) (*time.Time, error) {
	if start == nil {
		return nil, errNilSuppressionStart
	}
	if duration == nil {
		return nil, errNilSuppressionDuration
	}
	expiry := start.Truncate(time.Second).Add(duration.Truncate(time.Second))
	return &expiry, nil
}

func (ds *datastoreImpl) updateCache(vulns ...*storage.ImageCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, vuln := range vulns {
		// Vulnerabilities are snoozed by cve (name) and not by ID for backward compatibility purpose (when cve name and id were same).
		ds.cveSuppressionCache[vuln.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(vuln)
	}
}

func (ds *datastoreImpl) deleteFromCache(vulns ...*storage.ImageCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, vuln := range vulns {
		delete(ds.cveSuppressionCache, vuln.GetCveBaseInfo().GetCve())
	}
}

func gatherKeys(vulns []*storage.ImageCVE) [][]byte {
	keys := make([][]byte, 0, len(vulns))
	for _, vuln := range vulns {
		keys = append(keys, []byte(vuln.GetId()))
	}
	return keys
}

func convertMany(cves []*storage.ImageCVE, results []pkgSearch.Result) ([]*v1.SearchResult, error) {
	if len(cves) != len(results) {
		return nil, errors.Errorf("expected %d CVEs but got %d", len(results), len(cves))
	}

	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(cve *storage.ImageCVE, result *pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_VULNERABILITIES,
		Id:             cve.GetId(),
		Name:           cve.GetCveBaseInfo().GetCve(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
