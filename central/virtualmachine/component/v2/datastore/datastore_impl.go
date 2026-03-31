package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	pgStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage pgStore.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "Search")
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "Count")
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchRawVMComponents(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachineComponentV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "SearchRawVMComponents")
	var components []*storage.VirtualMachineComponentV2
	err := ds.storage.GetByQueryFn(ctx, q, func(component *storage.VirtualMachineComponentV2) error {
		components = append(components, component)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.VirtualMachineComponentV2, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "Get")
	component, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return component, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "Exists")
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.VirtualMachineComponentV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineComponentV2", "GetBatch")
	components, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}
