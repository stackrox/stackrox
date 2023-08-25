package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/reports/snapshot/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/snapshot/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	resourceType = "ReportSnapshot"
)

var (
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)
)

type datastoreImpl struct {
	storage  pgStore.Store
	searcher search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Search")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Count")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return 0, err
	}
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchReportSnapshots(ctx context.Context, q *v1.Query) ([]*storage.ReportSnapshot, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "SearchReportSnapshots")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.SearchReportSnapshots(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "SearchResults")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ReportSnapshot, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Get")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}
	snap, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return snap, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Exists")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return false, err
	}
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ReportSnapshot, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "GetMany")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	snaps, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return snaps, nil
}

func (ds *datastoreImpl) AddReportSnapshot(ctx context.Context, snap *storage.ReportSnapshot) (string, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "AddReportSnapshot")
	if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
		return "", err
	}
	if snap.ReportId != "" {
		return "", errors.New("New report snapshot must have an empty report id")
	}
	snap.ReportId = uuid.NewV4().String()
	if err := ds.storage.Upsert(ctx, snap); err != nil {
		return "", err
	}
	return snap.ReportId, nil
}

func (ds *datastoreImpl) UpdateReportSnapshot(ctx context.Context, snap *storage.ReportSnapshot) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpdateReportSnapshot")
	if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if snap.ReportId == "" {
		return errors.New("Report snapshot must have a non-empty report id")
	}
	if err := ds.storage.Upsert(ctx, snap); err != nil {
		return err
	}
	return nil
}

func (ds *datastoreImpl) DeleteReportSnapshot(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DeleteReportSnapshot")
	if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := ds.storage.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (ds *datastoreImpl) Walk(ctx context.Context, fn func(report *storage.ReportSnapshot) error) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "Walk")
	if ok, err := workflowSAC.ReadAllowed(ctx); !ok || err != nil {
		return err
	}
	return ds.storage.Walk(ctx, fn)
}
