package datastore

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	virtualMachineStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	defaultPageSize = 100
	preAllocateCap  = math.MaxUint16
)

type datastoreImpl struct {
	store virtualMachineStore.VirtualMachineStore

	mutex sync.Mutex
}

func newDatastoreImpl(store virtualMachineStore.VirtualMachineStore) DataStore {
	ds := &datastoreImpl{
		store: store,
	}
	return ds
}

// CountVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) CountVirtualMachines(ctx context.Context, query *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "CountVirtualMachines")
	if query == nil {
		query = search.EmptyQuery()
	}

	return ds.store.Count(ctx, query)
}

// GetVirtualMachine delegates to the underlying store.
func (ds *datastoreImpl) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "GetVirtualMachine")

	return ds.store.Get(ctx, id)
}

// UpsertVirtualMachine sets the virtualMachine in the underlying data structure.
func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpsertVirtualMachine")

	if virtualMachine.GetId() == "" {
		return errors.New("cannot upsert a virtualMachine without an id")
	}

	now := time.Now()
	virtualMachine.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&now)

	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	oldVM, found, err := ds.GetVirtualMachine(ctx, virtualMachine.GetId())
	if err != nil {
		return errors.Wrap(err, "retrieving old virtual machine")
	}
	if found && oldVM != nil {
		// Propagate previous scan information to updated virtual machine
		virtualMachine.Scan = oldVM.GetScan()
	}

	return ds.store.UpsertMany(ctx, []*storage.VirtualMachine{virtualMachine})
}

func (ds *datastoreImpl) UpdateVirtualMachineScan(
	ctx context.Context,
	virtualMachineID string,
	scanData *storage.VirtualMachineScan,
) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpdateVirtualMachineScan")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	vmToUpdate, found, err := ds.store.Get(ctx, virtualMachineID)
	if err != nil {
		return errors.Wrap(err, "retrieving virtual machine for scan update")
	}
	if !found {
		return errox.NotFound
	}
	vmToUpdate.Scan = scanData

	return ds.store.UpsertMany(ctx, []*storage.VirtualMachine{vmToUpdate})
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "DeleteVirtualMachines")

	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	return ds.store.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Exists")
	return ds.store.Exists(ctx, id)
}

func (ds *datastoreImpl) SearchRawVirtualMachines(
	ctx context.Context,
	query *v1.Query,
) ([]*storage.VirtualMachine, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "SearchRawVirtualMachines")
	// Sort by default by virtual machine name and namespace (does not apply if sort options are provided).
	// TODO(ROX-31024): Move default sorting over multiple columns to store
	searchQuery := query.CloneVT()
	if len(searchQuery.GetPagination().GetSortOptions()) == 0 {
		if searchQuery.GetPagination() == nil {
			searchQuery.Pagination = &v1.QueryPagination{}
		}
		searchQuery.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.VirtualMachineName.String(),
			},
			{
				Field: search.Namespace.String(),
			},
		}
	}
	pageSize := searchQuery.GetPagination().GetLimit()
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	// Limit pre-allocation size (some code paths set the pagination limit to
	// math.MaxInt32) and risk of OOMKills.
	if pageSize > preAllocateCap {
		pageSize = preAllocateCap
	}
	results := make([]*storage.VirtualMachine, 0, pageSize)
	err := ds.store.WalkByQuery(ctx, searchQuery, func(vm *storage.VirtualMachine) error {
		results = append(results, vm)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Walk iterates over all virtual machines and invokes the provided function for each.
// This method is optimized for processing VMs without loading them all into memory.
func (ds *datastoreImpl) Walk(ctx context.Context, fn func(vm *storage.VirtualMachine) error) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Walk")
	return ds.store.Walk(ctx, fn, true)
}
