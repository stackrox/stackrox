package datastore

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

const (
	defaultPageSize = 100
	preAllocateCap  = math.MaxUint16
)

type datastoreImpl struct {
	store store.Store
}

func newDatastoreImpl(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

func (ds *datastoreImpl) CountVirtualMachines(ctx context.Context, query *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "CountVirtualMachines")
	if query == nil {
		query = search.EmptyQuery()
	}
	return ds.store.Count(ctx, query)
}

func (ds *datastoreImpl) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachineV2, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "GetVirtualMachine")
	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) GetManyVirtualMachines(ctx context.Context, ids []string) ([]*storage.VirtualMachineV2, []int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "GetManyVirtualMachines")
	return ds.store.GetMany(ctx, ids)
}

func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, vm *storage.VirtualMachineV2) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "UpsertVirtualMachine")

	if vm.GetId() == "" {
		return errors.New("cannot upsert a virtual machine without an id")
	}
	return ds.store.UpsertVM(ctx, vm)
}

func (ds *datastoreImpl) UpsertScan(ctx context.Context, vmID string, parts common.VMScanParts) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "UpsertScan")

	if vmID == "" {
		return errors.New("cannot upsert scan without a VM id")
	}
	return ds.store.UpsertScan(ctx, vmID, parts)
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "DeleteVirtualMachines")
	return ds.store.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "Search")
	return ds.store.Search(ctx, query)
}

func (ds *datastoreImpl) SearchRawVirtualMachines(ctx context.Context, query *v1.Query) ([]*storage.VirtualMachineV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "SearchRawVirtualMachines")

	searchQuery := query.CloneVT()
	if len(searchQuery.GetPagination().GetSortOptions()) == 0 {
		if searchQuery.GetPagination() == nil {
			searchQuery.Pagination = &v1.QueryPagination{}
		}
		searchQuery.Pagination.SortOptions = []*v1.QuerySortOption{
			{Field: search.VirtualMachineName.String()},
			{Field: search.Namespace.String()},
		}
	}
	pageSize := searchQuery.GetPagination().GetLimit()
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > preAllocateCap {
		pageSize = preAllocateCap
	}
	results := make([]*storage.VirtualMachineV2, 0, pageSize)
	err := ds.store.WalkByQuery(ctx, searchQuery, func(vm *storage.VirtualMachineV2) error {
		results = append(results, vm)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (ds *datastoreImpl) Walk(ctx context.Context, fn func(vm *storage.VirtualMachineV2) error) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachineV2", "Walk")
	return ds.store.Walk(ctx, fn)
}
