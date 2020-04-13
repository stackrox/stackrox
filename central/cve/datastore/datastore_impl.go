package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
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
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}
	cve, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	found, err := ds.storage.Exists(id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	cves, _, err := ds.storage.GetBatch(ids)
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
			currentIndex++
		}
	}

	// Store the new CVE data.
	return ds.storage.UpsertClusterCVEs(parts...)
}

func (ds *datastoreImpl) Suppress(ctx context.Context, ids ...string) error {
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
		cve.Suppressed = true
	}
	return ds.storage.Upsert(cves...)
}

func (ds *datastoreImpl) Unsuppress(ctx context.Context, ids ...string) error {
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
	}
	return ds.storage.Upsert(cves...)
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Delete(ids...)
}
