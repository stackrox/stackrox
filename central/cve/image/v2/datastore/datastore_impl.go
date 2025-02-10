package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/search"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
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
	storage  store.Store
	searcher search.Searcher

	cveSuppressionLock  sync.RWMutex
	cveSuppressionCache common.CVESuppressionCache

	keyFence concurrency.KeyFence
}

//func getSuppressionCacheEntry(cve *storage.ImageCVEV2) common.SuppressionCacheEntry {
//	cacheEntry := common.SuppressionCacheEntry{}
//	cacheEntry.SuppressActivation = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeStart())
//	cacheEntry.SuppressExpiry = protocompat.ConvertTimestampToTimeOrNil(cve.GetSnoozeExpiry())
//	return cacheEntry
//}

//func (ds *datastoreImpl) buildSuppressedCache() {
//	query := pkgSearch.NewQueryBuilder().AddBools(pkgSearch.CVESuppressed, true).ProtoQuery()
//	suppressedCVEs, err := ds.searcher.SearchRawImageCVEs(accessAllCtx, query)
//	if err != nil {
//		log.Error(errors.Wrap(err, "Vulnerability exception management may not function correctly. Failed to build cache of CVE exceptions."))
//		return
//	}
//
//	ds.cveSuppressionLock.Lock()
//	defer ds.cveSuppressionLock.Unlock()
//	for _, cve := range suppressedCVEs {
//		ds.cveSuppressionCache[cve.GetCveBaseInfo().GetCve()] = getSuppressionCacheEntry(cve)
//	}
//}

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
