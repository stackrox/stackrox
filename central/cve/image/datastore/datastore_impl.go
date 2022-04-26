package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/image/datastore/internal/search"
	"github.com/stackrox/rox/central/cve/image/datastore/internal/store"
	"github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	pkgPostgres "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vulnRequesterOrApproverSAC = sac.ForResources(
		sac.ForResource(resources.VulnerabilityManagementRequests),
		sac.ForResource(resources.VulnerabilityManagementApprovals),
	)

	accessAllCtx = sac.WithAllAccess(context.Background())
)

type imageCVEPks struct {
	cve string
	os  string
}

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

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

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchCVEs(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
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
	pks, err := getPKs(id)
	if err != nil {
		return nil, false, err
	}
	cve, found, err := ds.storage.Get(ctx, id, pks.cve, pks.os)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	pks, err := getPKs(id)
	if err != nil {
		return false, err
	}
	found, err := ds.storage.Exists(ctx, id, pks.cve, pks.os)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
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
	if err := ds.storage.UpsertMany(ctx, cves); err != nil {
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
	if err := ds.storage.UpsertMany(ctx, cves); err != nil {
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

func getSuppressExpiry(start *types.Timestamp, duration *types.Duration) (*types.Timestamp, error) {
	d, err := types.DurationFromProto(duration)
	if err != nil || d == 0 {
		return nil, err
	}
	return &types.Timestamp{Seconds: start.GetSeconds() + int64(d.Seconds())}, nil
}

func getPKs(id string) (imageCVEPks, error) {
	parts := pkgPostgres.IDToParts(id)
	if len(parts) != 2 {
		return imageCVEPks{}, errors.Errorf("unexpected number of primary keys (%v) found for image cves. Expected 2 parts", parts)
	}

	return imageCVEPks{
		cve: parts[0],
		os:  parts[1],
	}, nil
}
