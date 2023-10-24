package datastore

import (
	"context"

	"github.com/cloudflare/cfssl/log"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/datastore/search"
	"github.com/stackrox/rox/central/cve/image/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache

	keyFence concurrency.KeyFence
}

func (ds *datastoreImpl) buildSuppressedCache() {
	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
	suppressedCVEs, err := ds.searcher.SearchRawImageCVEs(accessAllCtx, query)
	if err != nil {
		log.Error(errors.Wrap(err, "Vulnerability exception management may not function correctly. Failed to build cache of CVE exceptions."))
		return
	}

	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()
	for _, cve := range suppressedCVEs {
		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = common.SuppressionCacheEntry{
			SuppressActivation: cve.GetSnoozeStart(),
			SuppressExpiry:     cve.GetSnoozeExpiry(),
		}
	}
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchImageCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVE, error) {
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

	vulns, err := ds.searcher.SearchRawImageCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
	if err != nil {
		return err
	}

	err = ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(gatherKeys(vulns)...), func() error {
		for _, vuln := range vulns {
			vuln.Snoozed = true
			vuln.SnoozeStart = start
			vuln.SnoozeExpiry = expiry
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

	vulns, err := ds.searcher.SearchRawImageCVEs(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, cves...).ProtoQuery())
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

func (ds *datastoreImpl) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	ds.cveSuppressionLock.RLock()
	defer ds.cveSuppressionLock.RUnlock()

	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if entry, ok := ds.cveSuppressionCache[vuln.GetCve()]; ok {
				vuln.Suppressed = true
				vuln.SuppressActivation = entry.SuppressActivation
				vuln.SuppressExpiry = entry.SuppressExpiry
				vuln.State = storage.VulnerabilityState_DEFERRED
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

func (ds *datastoreImpl) updateCache(vulns ...*storage.ImageCVE) {
	ds.cveSuppressionLock.Lock()
	defer ds.cveSuppressionLock.Unlock()

	for _, vuln := range vulns {
		// Vulnerabilities are snoozed by cve (name) and not by ID for backward compatibility purpose (when cve name and id were same).
		ds.cveSuppressionCache[vuln.GetCveBaseInfo().GetCve()] = common.SuppressionCacheEntry{
			SuppressActivation: vuln.SnoozeStart,
			SuppressExpiry:     vuln.SnoozeExpiry,
		}
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
