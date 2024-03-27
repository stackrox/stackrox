package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/cluster/datastore/search"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/converter/v2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	clusterSAC = sac.ForResource(resources.Cluster)

	accessAllCtx = sac.WithAllAccess(context.Background())

	errNilSuppressionStart = errors.New("suppression start time is nil")

	errNilSuppressionDuration = errors.New("suppression duration is nil")
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache
}

func (ds *datastoreImpl) UpsertClusterCVEsInternal(ctx context.Context, cveType storage.CVE_CVEType, cveParts ...converter.ClusterCVEParts) error {
	if ok, err := clusterSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.ReconcileClusterCVEParts(ctx, cveType, cveParts...)
}

func (ds *datastoreImpl) DeleteClusterCVEsInternal(ctx context.Context, clusterID string) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.DeleteClusterCVEsForCluster(ctx, clusterID)
}

func getSuppressionCacheEntry(cve *storage.ClusterCVE) common.SuppressionCacheEntry {
	cacheEntry := common.SuppressionCacheEntry{}
	cacheEntry.SuppressActivation = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeStart())
	if cve.GetSnoozeStart() != nil {
		suppressActivation, _ := protocompat.ConvertTimestampToTimeOrError(cve.GetSnoozeStart())
		cacheEntry.SuppressActivation = &suppressActivation
	}
	if cve.GetSnoozeExpiry() != nil {
		suppressExpiry, _ := protocompat.ConvertTimestampToTimeOrError(cve.GetSnoozeExpiry())
		cacheEntry.SuppressExpiry = &suppressExpiry
	}
	return cacheEntry
}

func (ds *datastoreImpl) buildSuppressedCache() error {
	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.searcher.SearchRawClusterCVEs(accessAllCtx, query)
	if err != nil {
		return errors.Wrap(err, "searching suppress CVEs")
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range suppressedCVEs {
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(cve)
	}
	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchClusterCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchClusterCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVE, error) {
	cves, err := ds.searcher.SearchRawClusterCVEs(ctx, q)
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

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ClusterCVE, bool, error) {
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ClusterCVE, error) {
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

	vulns, err := ds.searcher.SearchRawClusterCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	for _, vuln := range vulns {
		vuln.Snoozed = true
		vuln.SnoozeStart = protocompat.ConvertTimeToTimestampOrNil(start)
		vuln.SnoozeExpiry = protocompat.ConvertTimeToTimestampOrNil(expiry)
	}
	if err := ds.storage.UpsertMany(ctx, vulns); err != nil {
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

	vulns, err := ds.searcher.SearchRawClusterCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	for _, vuln := range vulns {
		vuln.Snoozed = false
		vuln.SnoozeStart = nil
		vuln.SnoozeExpiry = nil
	}
	if err := ds.storage.UpsertMany(ctx, vulns); err != nil {
		return err
	}

	ds.deleteFromCache(vulns...)
	return nil
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

func (ds *datastoreImpl) updateCache(cves ...*storage.ClusterCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		// Vulnerabilities are snoozed by cve (name) and not by ID field to for backward compatibility purpose (when cve name and id were same).
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(cve)
	}
}

func (ds *datastoreImpl) deleteFromCache(cves ...*storage.ClusterCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		delete(ds.cveSuppressionCache, cve.GetCveBaseInfo().GetCve())
	}
}
