package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/node/datastore/store"
	"github.com/stackrox/rox/central/metrics"
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

	nodeSAC = sac.ForResource(resources.Node)

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

func getSuppressionCacheEntry(cve *storage.NodeCVE) common.SuppressionCacheEntry {
	cacheEntry := common.SuppressionCacheEntry{}
	cacheEntry.SuppressActivation = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeStart())
	cacheEntry.SuppressExpiry = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeExpiry())
	return cacheEntry
}

func (ds *datastoreImpl) buildSuppressedCache() error {
	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.SearchRawCVEs(accessAllCtx, query)
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
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
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

func (ds *datastoreImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error) {
	var cves []*storage.NodeCVE
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.NodeCVE) error {
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

func (ds *datastoreImpl) UpsertMany(ctx context.Context, cves []*storage.NodeCVE) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "NodeCVE", "UpsertMany")

	if ok, err := nodeSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.UpsertMany(ctx, cves); err != nil {
		return errors.Wrap(err, "Upserting node CVEs")
	}
	return nil
}

func (ds *datastoreImpl) PruneNodeCVEs(ctx context.Context, ids []string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "NodeCVE", "PruneNodeCVEs")

	if ok, err := nodeSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.PruneMany(ctx, ids); err != nil {
		return errors.Wrap(err, "Pruning node CVEs")
	}
	return nil
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

	vulns, err := ds.SearchRawCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
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

	vulns, err := ds.SearchRawCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
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

func (ds *datastoreImpl) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()

	for _, component := range node.GetScan().GetComponents() {
		for _, vuln := range component.GetVulnerabilities() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCveBaseInfo().GetCve()]; ok {
				vuln.Snoozed = true
				vuln.SnoozeStart = protocompat.ConvertTimeToTimestampOrNil(entry.SuppressActivation)
				vuln.SnoozeExpiry = protocompat.ConvertTimeToTimestampOrNil(entry.SuppressExpiry)
			}
		}
	}
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

func (ds *datastoreImpl) updateCache(cves ...*storage.NodeCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		// Vulnerabilities are snoozed by cve (name) and not by ID field to for backward compatibility purpose (when cve name and id were same).
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(cve)
	}
}

func (ds *datastoreImpl) deleteFromCache(cves ...*storage.NodeCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, cve := range cves {
		delete(ds.cveSuppressionCache, cve.GetCveBaseInfo().GetCve())
	}
}

func gatherKeys(vulns []*storage.NodeCVE) [][]byte {
	keys := make([][]byte, 0, len(vulns))
	for _, vuln := range vulns {
		keys = append(keys, []byte(vuln.GetId()))
	}
	return keys
}

func convertMany(cves []*storage.NodeCVE, results []pkgSearch.Result) ([]*v1.SearchResult, error) {
	if len(cves) != len(results) {
		return nil, errors.Errorf("expected %d CVEs, got %d", len(results), len(cves))
	}

	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(cve *storage.NodeCVE, result *pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_VULNERABILITIES,
		Id:             cve.GetId(),
		Name:           cve.GetCveBaseInfo().GetCve(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
