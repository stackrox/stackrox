package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/cve/common"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/index"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/search"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/store"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	vulnRequesterOrApproverSAC = sac.ForResources(
		sac.ForResource(resources.VulnerabilityManagementRequests),
		sac.ForResource(resources.VulnerabilityManagementApprovals),
	)

	accessAllCtx = sac.WithAllAccess(context.Background())
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache
}

func (ds *datastoreImpl) buildSuppressedCache() error {
	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.searcher.SearchRawCVEs(accessAllCtx, query)
	if err != nil {
		return errors.Wrap(err, "searching suppress CVEs")
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range suppressedCVEs {
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = common.SuppressionCacheEntry{
			SuppressActivation: cve.GetSnoozeStart(),
			SuppressExpiry:     cve.GetSnoozeExpiry(),
		}
	}
	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error) {
	cves, err := ds.searcher.SearchRawCVEs(ctx, q)
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

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error) {
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.NodeCVE, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error {
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	expiry, err := getSuppressExpiry(start, duration)
	if err != nil {
		return err
	}

	vulns, err := ds.searcher.SearchRawCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	for _, vuln := range vulns {
		vuln.Snoozed = true
		vuln.SnoozeStart = start
		vuln.SnoozeExpiry = expiry
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

	vulns, err := ds.searcher.SearchRawCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
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

func (ds *datastoreImpl) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()

	for _, component := range node.GetScan().GetComponents() {
		for _, vuln := range component.GetVulnerabilities() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCveBaseInfo().GetCve()]; ok {
				vuln.Snoozed = true
				vuln.SnoozeStart = entry.SuppressActivation
				vuln.SnoozeExpiry = entry.SuppressExpiry
			}
		}
	}
}

func getSuppressExpiry(start *types.Timestamp, duration *types.Duration) (*types.Timestamp, error) {
	d, err := types.DurationFromProto(duration)
	if err != nil || d == 0 {
		return nil, err
	}
	return &types.Timestamp{Seconds: start.GetSeconds() + int64(d.Seconds())}, nil
}

func (ds *datastoreImpl) updateCache(cves ...*storage.NodeCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		// Vulnerabilities are snoozed by cve (name) and not by ID field to for backward compatibility purpose (when cve name and id were same).
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = common.SuppressionCacheEntry{
			SuppressActivation: cve.SnoozeStart,
			SuppressExpiry:     cve.SnoozeExpiry,
		}
	}
}

func (ds *datastoreImpl) deleteFromCache(cves ...*storage.NodeCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		delete(ds.cveSuppressionCache, cve.GetCveBaseInfo().GetCve())
	}
}
