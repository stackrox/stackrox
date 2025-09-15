package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	virtualMachineStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

const (
	defaultResultSize = 1000
	defaultPageSize = 100
)

type datastoreImpl struct {
	store virtualMachineStore.VirtualMachineStore
}

func newDatastoreImpl(store virtualMachineStore.VirtualMachineStore) DataStore {
	ds := &datastoreImpl{
		store: store,
	}
	return ds
}

// CountVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) CountVirtualMachines(ctx context.Context) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "CountVirtualMachines")

	return ds.store.Count(ctx, search.EmptyQuery())
}

// GetVirtualMachine delegates to the underlying store.
func (ds *datastoreImpl) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "GetVirtualMachine")

	return ds.store.Get(ctx, id)
}

// GetAllVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) GetAllVirtualMachines(ctx context.Context) ([]*storage.VirtualMachine, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "GetAllVirtualMachines")

	ret := make([]*storage.VirtualMachine, 0, defaultResultSize)
	err := ds.store.Walk(ctx, func(vm *storage.VirtualMachine) error {
		ret = append(ret, vm)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// UpsertVirtualMachine sets the virtualMachine in the underlying data structure.
func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpsertVirtualMachine")

	if virtualMachine.GetId() == "" {
		return errors.New("cannot upsert a virtualMachine without an id")
	}

	now := time.Now()
	virtualMachine.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&now)

	return ds.store.UpsertMany(ctx, []*storage.VirtualMachine{virtualMachine})
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "DeleteVirtualMachines")

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
	pageSize := query.GetPagination().GetLimit()
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	results := make([]*storage.VirtualMachine, 0, pageSize)
	err := ds.store.WalkByQuery(ctx, query, func(vm *storage.VirtualMachine) error {
		results = append(results, vm)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}
