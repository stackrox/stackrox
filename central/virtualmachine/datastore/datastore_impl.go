package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	virtualMachineStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vmSAC = sac.ForResource(resources.VirtualMachine)
)

type datastoreImpl struct {
	mutex sync.RWMutex
	store virtualMachineStore.VirtualMachineStore
}

func newDatastoreImpl(store virtualMachineStore.VirtualMachineStore) DataStore {
	ds := &datastoreImpl{
		mutex: sync.RWMutex{},
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

	ret := make([]*storage.VirtualMachine, 0, 10)
	err := ds.store.Walk(ctx, func(vm *storage.VirtualMachine) error {
		ret = append(ret, vm)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// CreateVirtualMachine works like upsert except it rejects requests for VMs that already exist in the underlying data structure
func (ds *datastoreImpl) CreateVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "CreateVirtualMachine")

	allowed, err := vmSAC.WriteAllowed(ctx)
	if err != nil {
		return err
	} else if !allowed {
		return sac.ErrResourceAccessDenied
	}

	if virtualMachine.GetId() == "" {
		return errors.New("cannot create a virtualMachine without an id")
	}

	exists := false
	concurrency.WithLock(&ds.mutex, func() {
		exists, err = ds.store.Exists(ctx, virtualMachine.GetId())
		if err != nil || exists {
			return
		}
		err = ds.store.UpsertMany(ctx, []*storage.VirtualMachine{virtualMachine})
	})
	if err != nil {
		return err
	}
	if exists {
		return errors.New("Already exists")
	}
	return nil
}

// UpsertVirtualMachine sets the virtualMachine in the underlying data structure.
func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpsertVirtualMachine")

	if virtualMachine.GetId() == "" {
		return errors.New("cannot upsert a virtualMachine without an id")
	}

	now := time.Now()
	virtualMachine.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&now)

	var err error
	concurrency.WithLock(&ds.mutex, func() {
		err = ds.store.UpsertMany(ctx, []*storage.VirtualMachine{virtualMachine})
	})
	return err
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "DeleteVirtualMachines")

	return ds.store.DeleteMany(ctx, ids)
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Exists")
	return ds.store.Exists(ctx, id)
}
