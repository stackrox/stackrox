package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	virtualMachinesSAC = sac.ForResource(resources.VirtualMachine)
	allAccessCtx       = sac.WithAllAccess(context.Background())
)

type datastoreImpl struct {
	keyedMutex      *concurrency.KeyedMutex
	virtualMachines []*storage.VirtualMachine
}

func newDatastoreImpl() DataStore {
	ds := &datastoreImpl{
		keyedMutex:      concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		virtualMachines: []*storage.VirtualMachine{},
	}
	return ds
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Search")
	return []pkgSearch.Result{}, nil
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Count")
	return len(ds.virtualMachines), nil
}

func (ds *datastoreImpl) SearchVirtualMachines(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "SearchVirtualMachines")
	return []*v1.SearchResult{}, nil
}

// SearchRawVirtualMachines delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawVirtualMachines(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachine, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "SearchRawVirtualMachines")
	return ds.virtualMachines, nil
}

// CountVirtualMachines delegates to the underlying store.
func (ds *datastoreImpl) CountVirtualMachines(ctx context.Context) (int, error) {
	if _, err := virtualMachinesSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	}
	return len(ds.virtualMachines), nil
}

// GetVirtualMachine delegates to the underlying store.
func (ds *datastoreImpl) GetVirtualMachine(ctx context.Context, id string) (*storage.VirtualMachine, bool, error) {
	for _, vm := range ds.virtualMachines {
		if vm.Id == id {
			return vm, true, nil
		}
	}
	return nil, false, nil
}

// UpsertVirtualMachine dedupes the virtualMachine with the underlying storage and adds the virtualMachine to the index.
func (ds *datastoreImpl) UpsertVirtualMachine(ctx context.Context, virtualMachine *storage.VirtualMachine) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "UpsertVirtualMachine")

	if virtualMachine.GetId() == "" {
		return errors.New("cannot upsert a virtualMachine without an id")
	}

	if ok, err := virtualMachinesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(virtualMachine.GetId())
	defer ds.keyedMutex.Unlock(virtualMachine.GetId())

	existingIndex := -1
	for i, vm := range ds.virtualMachines {
		if vm.Id == virtualMachine.Id {
			existingIndex = i
			break
		}
	}

	if existingIndex < 0 {
		ds.virtualMachines = append(ds.virtualMachines, virtualMachine)
	} else {
		ds.virtualMachines[existingIndex] = virtualMachine
	}

	return nil
}

func (ds *datastoreImpl) DeleteVirtualMachines(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "DeleteVirtualMachines")

	if ok, err := virtualMachinesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	var toDelete []int
	for _, id := range ids {
		for i, vm := range ds.virtualMachines {
			if vm.Id == id {
				toDelete = append(toDelete, i)
				break
			}
		}
	}

	var newVirtualMachines []*storage.VirtualMachine
	for i, vm := range ds.virtualMachines {
		thisVmShouldBeDeleted := false
		for _, toDeleteidx := range toDelete {
			if i == toDeleteidx {
				thisVmShouldBeDeleted = true
			}
		}

		if !thisVmShouldBeDeleted {
			newVirtualMachines = append(newVirtualMachines, vm)
		}
	}

	return nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "VirtualMachine", "Exists")

	for _, vm := range ds.virtualMachines {
		if vm.Id == id {
			return true, nil
		}
	}
	return false, nil
}
