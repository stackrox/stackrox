package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	pgStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage pgStore.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "Search")
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "Count")
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchRawVMScans(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachineScanV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "SearchRawVMScans")
	var scans []*storage.VirtualMachineScanV2
	err := ds.storage.GetByQueryFn(ctx, q, func(scan *storage.VirtualMachineScanV2) error {
		scans = append(scans, scan)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return scans, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.VirtualMachineScanV2, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "Get")
	scan, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return scan, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "Exists")
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.VirtualMachineScanV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineScanV2", "GetBatch")
	scans, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return scans, nil
}
