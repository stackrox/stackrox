package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	pgStore "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage pgStore.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "Search")
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "Count")
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchRawVMCVEs(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachineCVEV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "SearchRawVMCVEs")
	var cves []*storage.VirtualMachineCVEV2
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.VirtualMachineCVEV2) error {
		cves = append(cves, cve)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.VirtualMachineCVEV2, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "Get")
	cve, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "Exists")
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.VirtualMachineCVEV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineCVEV2", "GetBatch")
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}
