package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/index"
	sacFilters "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vulnRequesterOrApproverSAC = sac.ForResources(
		sac.ForResource(resources.VulnerabilityManagementRequests),
		sac.ForResource(resources.VulnerabilityManagementApprovals),
	)
	clustersSAC = sac.ForResource(resources.Cluster)

	accessAllCtx = sac.WithAllAccess(context.Background())
)

type datastoreImpl struct {
	storage       store.Store
	indexer       index.Indexer
	searcher      search.Searcher
	graphProvider graph.Provider

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache map[string]suppressionCacheEntry
}

type suppressionCacheEntry struct {
	Suppressed         bool
	SuppressActivation *types.Timestamp
	SuppressExpiry     *types.Timestamp
}

func (ds *datastoreImpl) buildSuppressedCache() error {
	query := searchPkg.NewQueryBuilder().AddBools(searchPkg.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.searcher.SearchRawCVEs(accessAllCtx, query)
	if err != nil {
		return errors.Wrap(err, "searching suppress CVEs")
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range suppressedCVEs {
		ds.cveSuppressionCache[cve.GetId()] = suppressionCacheEntry{
			Suppressed:         cve.GetSuppressed(),
			SuppressActivation: cve.GetSuppressActivation(),
			SuppressExpiry:     cve.GetSuppressExpiry(),
		}
	}
	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	cves, err := ds.searcher.SearchRawCVEs(ctx, q)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.CVE, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	cve, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return false, err
	}

	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	filteredIDs, err := ds.filterReadable(ctx, ids)
	if err != nil {
		return nil, err
	}

	cves, _, err := ds.storage.GetMany(ctx, filteredIDs)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error {
	// Image permission check replaced by vuln request permission check in 68.0. Previously, global image permission
	// check did not guarantee that the requestor is disallowed from suppressing node/cluster cves. Hence, verification
	// to determine if the cve is image cve was not added.
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	expiry, err := getSuppressExpiry(start, duration)
	if err != nil {
		return err
	}

	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return err
	}

	for _, cve := range cves {
		cve.Suppressed = true
		cve.SuppressActivation = start
		cve.SuppressExpiry = expiry
	}
	if err := ds.storage.Upsert(ctx, cves...); err != nil {
		return err
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range cves {
		ds.cveSuppressionCache[cve.GetId()] = suppressionCacheEntry{
			Suppressed:         cve.Suppressed,
			SuppressActivation: cve.SuppressActivation,
			SuppressExpiry:     cve.SuppressExpiry,
		}
	}
	return nil
}

func (ds *datastoreImpl) Unsuppress(ctx context.Context, ids ...string) error {
	// Image permission check replaced by vuln request permission check in 68.0. Previously, global image permission
	// check did not guarantee that the requestor is disallowed from unsuppressing node/cluster cves. Hence, verification
	// to determine if the cve is image cve was not added.
	if ok, err := vulnRequesterOrApproverSAC.WriteAllowedToAll(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return err
	}

	for _, cve := range cves {
		cve.Suppressed = false
		cve.SuppressActivation = nil
		cve.SuppressExpiry = nil
	}
	if err := ds.storage.Upsert(ctx, cves...); err != nil {
		return err
	}
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range cves {
		delete(ds.cveSuppressionCache, cve.GetId())
	}
	return nil
}

func (ds *datastoreImpl) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCve()]; ok {
				vuln.Suppressed = entry.Suppressed
				vuln.SuppressActivation = entry.SuppressActivation
				vuln.SuppressExpiry = entry.SuppressExpiry

				vuln.State = storage.VulnerabilityState_DEFERRED
			}
		}
	}
}

func (ds *datastoreImpl) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()
	for _, component := range node.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCve()]; ok {
				vuln.Suppressed = entry.Suppressed
				vuln.SuppressActivation = entry.SuppressActivation
				vuln.SuppressExpiry = entry.SuppressExpiry
			}
		}
	}
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if ok, err := clustersSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.Delete(ctx, ids...); err != nil {
		return err
	}
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, id := range ids {
		delete(ds.cveSuppressionCache, id)
	}
	return nil
}

func getSuppressExpiry(start *types.Timestamp, duration *types.Duration) (*types.Timestamp, error) {
	d, err := types.DurationFromProto(duration)
	if err != nil || d == 0 {
		return nil, err
	}
	return &types.Timestamp{Seconds: start.GetSeconds() + int64(d.Seconds())}, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
