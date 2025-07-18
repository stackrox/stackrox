package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	virtualMachinesSAC = sac.ForResource(resources.VirtualMachine)
)

type datastoreImpl struct {
	mutex           sync.RWMutex
	virtualMachines map[string]*storage.VirtualMachine
}

func newDatastoreImpl() DataStore {
	ds := &datastoreImpl{
		mutex:           sync.RWMutex{},
		virtualMachines: map[string]*storage.VirtualMachine{},
	}
	return ds
}

// CountVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) CountVirtualMachines(ctx context.Context) (int, error) {
	if ok, err := virtualMachinesSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, sac.ErrResourceAccessDenied
	}

	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	return len(ds.virtualMachines), nil
}

// GetVirtualMachine delegates to the underlying store.
func (ds *datastoreImpl) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error) {
	if ok, err := virtualMachinesSAC.ReadAllowed(ctx); err != nil {
		return &storage.VirtualMachine{}, false, err
	} else if !ok {
		return &storage.VirtualMachine{}, false, sac.ErrResourceAccessDenied
	}

	if id == "" {
		return nil, false, errors.New("Please specify an id")
	}

	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	vm, found := ds.virtualMachines[id]

	if found {
		cloned := vm.CloneVT()
		return cloned, true, nil
	}

	return nil, false, nil
}

// GetAllVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) GetAllVirtualMachines(ctx context.Context) ([]*storage.VirtualMachine, error) {
	if ok, err := virtualMachinesSAC.ReadAllowed(ctx); err != nil {
		return []*storage.VirtualMachine{}, err
	} else if !ok {
		return []*storage.VirtualMachine{}, sac.ErrResourceAccessDenied
	}

	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	ret := make([]*storage.VirtualMachine, 0, len(ds.virtualMachines))

	for _, v := range ds.virtualMachines {
		ret = append(ret, v.CloneVT())
	}

	return ret, nil
}

// UpsertVirtualMachine dedupes the virtualMachine with the underlying storage and adds the virtualMachine to the index.
func (ds *datastoreImpl) CreateVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "CreateVirtualMachine")

	if ok, err := virtualMachinesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if virtualMachine.GetId() == "" {
		return errors.New("cannot create a virtualMachine without an id")
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	if _, exists := ds.virtualMachines[virtualMachine.GetId()]; exists {
		return errors.New("Already exists")
	}

	ds.virtualMachines[virtualMachine.GetId()] = virtualMachine

	return nil
}

// UpsertVirtualMachine dedupes the virtualMachine with the underlying storage and adds the virtualMachine to the index.
func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpsertVirtualMachine")

	if ok, err := virtualMachinesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if virtualMachine.GetId() == "" {
		return errors.New("cannot upsert a virtualMachine without an id")
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.virtualMachines[virtualMachine.GetId()] = virtualMachine

	return nil
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "DeleteVirtualMachines")

	if ok, err := virtualMachinesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	missingIds := make([]string, 0)

	// First check which IDs exist
	for _, id := range ids {
		if _, exists := ds.virtualMachines[id]; !exists {
			missingIds = append(missingIds, id)
		}
	}

	if len(missingIds) > 0 {
		return errors.Errorf("The following virtual machines ids not found: %v", missingIds)
	}

	// Only proceed with deletion if all IDs exist
	for _, id := range ids {
		delete(ds.virtualMachines, id)
	}

	return nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Exists")
	if ok, err := virtualMachinesSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if !ok {
		return false, sac.ErrResourceAccessDenied
	}

	if id == "" {
		return false, errors.New("Please specify a valid id")
	}

	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	_, exists := ds.virtualMachines[id]
	return exists, nil
}
