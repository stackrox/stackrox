package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/index"
	sacFilters "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	imagesSAC = sac.ForResource(resources.Image)

	getCVECtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.CVE)))

	log = logging.LoggerForModule()
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
	suppressedCVEs, err := ds.searcher.SearchRawCVEs(getCVECtx, query)
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

func (ds *datastoreImpl) Count(ctx context.Context) (int, error) {
	results, err := ds.searcher.Search(ctx, searchPkg.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.CVE, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	cve, found, err := ds.storage.Get(id)
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

	found, err := ds.storage.Exists(id)
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

	cves, _, err := ds.storage.GetBatch(filteredIDs)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Upsert(ctx context.Context, cves ...*storage.CVE) error {
	if len(cves) == 0 {
		return nil
	}

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	// Load the suppressed value for any CVEs already present.
	ids := make([]string, 0, len(cves))
	for _, cve := range cves {
		ids = append(ids, cve.GetId())
	}
	currentCVEs, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return err
	}
	var currentIndex int
	for newIndex := 0; newIndex < len(cves) && currentIndex < len(currentCVEs); newIndex++ {
		if currentCVEs[currentIndex].GetId() == cves[newIndex].GetId() {
			cves[newIndex].Suppressed = currentCVEs[currentIndex].Suppressed
			cves[newIndex].SuppressActivation = currentCVEs[currentIndex].SuppressActivation
			cves[newIndex].SuppressExpiry = currentCVEs[currentIndex].SuppressExpiry
			currentIndex++
		}
	}

	// Store the new CVE data.
	return ds.storage.Upsert(cves...)
}

func (ds *datastoreImpl) UpsertClusterCVEs(ctx context.Context, parts ...converter.ClusterCVEParts) error {
	if len(parts) == 0 {
		return nil
	}

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	// Load the suppressed value for any CVEs already present.
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, p.CVE.GetId())
	}
	currentCVEs, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return err
	}
	var currentIndex int
	for newIndex := 0; newIndex < len(parts) && currentIndex < len(currentCVEs); newIndex++ {
		if currentCVEs[currentIndex].GetId() == parts[newIndex].CVE.GetId() {
			parts[newIndex].CVE.Suppressed = currentCVEs[currentIndex].Suppressed
			parts[newIndex].CVE.SuppressActivation = currentCVEs[currentIndex].SuppressActivation
			parts[newIndex].CVE.SuppressExpiry = currentCVEs[currentIndex].SuppressExpiry
			currentIndex++
		}
	}

	// Store the new CVE data.
	return ds.storage.UpsertClusterCVEs(parts...)
}

func (ds *datastoreImpl) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error {
	// Check global write permissions since this may effect images risk/visibility in in places the user does not have read access.
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	expiry, err := getSuppressExpiry(start, duration)
	if err != nil {
		return err
	}

	cves, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return err
	}

	for _, cve := range cves {
		cve.Suppressed = true
		cve.SuppressActivation = start
		cve.SuppressExpiry = expiry
	}
	if err := ds.storage.Upsert(cves...); err != nil {
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
	// Check global write permissions since this may effect images risk/visibility in in places the user does not have read access.
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	cves, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return err
	}

	for _, cve := range cves {
		cve.Suppressed = false
		cve.SuppressActivation = nil
		cve.SuppressExpiry = nil
	}
	if err := ds.storage.Upsert(cves...); err != nil {
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
			}
		}
	}
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.storage.Delete(ids...); err != nil {
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
		filteredIDs, err = filtered.ApplySACFilters(graphContext, ids, sacFilters.GetSACFilters()...)
	})
	return filteredIDs, err
}
