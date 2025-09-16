//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	vmStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestVirtualMachineDataStore(t *testing.T) {
	suite.Run(t, new(VirtualMachineDataStoreTestSuite))
}

type VirtualMachineDataStoreTestSuite struct {
	suite.Suite

	db *pgtest.TestPostgres

	datastore DataStore
	ctx       context.Context
	sacCtx    context.Context
	noSacCtx  context.Context
}

func (s *VirtualMachineDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	store := vmStore.New(s.db)
	s.datastore = newDatastoreImpl(store)
	s.ctx = context.Background()
	s.sacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *VirtualMachineDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *VirtualMachineDataStoreTestSuite) createTestVM(id int) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		Id:        uuid.NewTestUUID(id).String(),
		Name:      fmt.Sprintf("test-vm-%d", id),
		Namespace: "default",
	}
}

// Test CountVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestCountVirtualMachines() {
	// Test with SAC allowed
	count, err := s.datastore.CountVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Equal(0, count)

	// Add a VM
	vm := s.createTestVM(1)
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	count, err = s.datastore.CountVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Equal(1, count)

	// Test with SAC denied
	// Namespace-scoped resource silently returns 0 if access is denied
	count, err = s.datastore.CountVirtualMachines(s.noSacCtx)
	s.NoError(err)
	s.Equal(0, count)
}

// Test GetVirtualMachine
func (s *VirtualMachineDataStoreTestSuite) TestGetVirtualMachine() {
	vm := s.createTestVM(1)
	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	nonExistentVMID := uuid.NewTestUUID(9).String()

	// Test successful get
	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(vm.GetId(), retrievedVM.GetId())
	s.Equal(vm.GetName(), retrievedVM.GetName())

	// Test not found
	retrievedVM, found, err = s.datastore.GetVirtualMachine(s.sacCtx, nonExistentVMID)
	s.NoError(err)
	s.False(found)
	s.Nil(retrievedVM)

	// Test empty ID
	retrievedVM, found, err = s.datastore.GetVirtualMachine(s.sacCtx, "")
	s.Error(err)
	s.False(found)
	s.Nil(retrievedVM)
	s.Contains(err.Error(), "ERROR: invalid input syntax for type uuid: \"\"")

	// Test SAC denied
	// Namespace-scoped resource:
	// Access to non-authorized object does not return any error, but
	// behaves for the requester as if the object did not exist.
	_, found, err = s.datastore.GetVirtualMachine(s.noSacCtx, vm.GetId())
	s.NoError(err)
	s.False(found)
}

// Test GetAllVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestGetAllVirtualMachines() {
	// Test empty store
	vms, err := s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Empty(vms)

	// Add multiple VMs
	vm1 := s.createTestVM(1)
	vm2 := s.createTestVM(2)

	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm1)
	s.NoError(err)
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm2)
	s.NoError(err)

	vms, err = s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Len(vms, 2)

	// Test SAC denied
	// Namespace-scoped resource:
	// Access to non-authorized objects does not return any error, but
	// behaves for the requester as if the requested objects don't exists.
	vms, err = s.datastore.GetAllVirtualMachines(s.noSacCtx)
	s.NoError(err)
	s.Empty(vms)
}

// Test UpsertVirtualMachine
func (s *VirtualMachineDataStoreTestSuite) TestUpsertVirtualMachine() {
	vm := s.createTestVM(1)

	// Test initial upsert (create)
	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Test update
	vm.Name = "updated-name"
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal("updated-name", retrievedVM.GetName())

	// Test empty ID
	vmNoId := &storage.VirtualMachine{Name: "test-vm-no-id"}
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vmNoId)
	s.Error(err)
	s.Contains(err.Error(), "cannot upsert a virtualMachine without an id")

	// Test SAC denied
	// Namespace-scoped resource
	// Access to non-authorized objects does not return any error but behaves
	// for the requester as if the objects do not exist.
	vm2 := s.createTestVM(2)
	err = s.datastore.UpsertVirtualMachine(s.noSacCtx, vm2)
	s.Error(err)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	retrievedVM2, foundVM2, err := s.datastore.GetVirtualMachine(s.sacCtx, vm2.GetId())
	s.NoError(err)
	s.False(foundVM2)
	s.Nil(retrievedVM2)
}

// Test DeleteVirtualMachines in one call
func (s *VirtualMachineDataStoreTestSuite) TestDeleteVirtualMachinesOneCall() {
	// Create test VMs
	vm1 := s.createTestVM(1)
	vmID1 := vm1.GetId()
	vm2 := s.createTestVM(2)
	vmID2 := vm2.GetId()

	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm1)
	s.NoError(err)
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm2)
	s.NoError(err)

	// Test successful deletion
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, vmID1, vmID2)
	s.NoError(err)

	// Verify deletion
	_, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vmID1)
	s.NoError(err)
	s.False(found)

	// Verify deletion
	_, found, err = s.datastore.GetVirtualMachine(s.sacCtx, vmID2)
	s.NoError(err)
	s.False(found)
}

// Test DeleteVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestDeleteVirtualMachines() {
	// Create test VMs
	vm1 := s.createTestVM(1)
	vmID1 := vm1.GetId()
	vm2 := s.createTestVM(2)
	vmID2 := vm2.GetId()

	nonExistentVMID := uuid.NewTestUUID(9).String()

	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm1)
	s.NoError(err)
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm2)
	s.NoError(err)

	// Test successful deletion
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, vmID1)
	s.NoError(err)

	// Verify deletion
	_, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vmID1)
	s.NoError(err)
	s.False(found)

	// Test deletion of non-existent VM
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, nonExistentVMID)
	s.NoError(err)

	// Test batch deletion with some missing
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, vmID2, nonExistentVMID)
	s.NoError(err)

	// Verify test-2 was removed
	_, found, err = s.datastore.GetVirtualMachine(s.sacCtx, vmID2)
	s.NoError(err)
	s.False(found)

	// Test SAC denied
	// Namespace-scoped resource
	// Access to non-authorized objects does not return any error, but behaves
	// for the requester as if the target objects do not exist.
	err = s.datastore.DeleteVirtualMachines(s.noSacCtx, vmID2)
	s.NoError(err)
}

// Test Exists
func (s *VirtualMachineDataStoreTestSuite) TestExists() {
	vm := s.createTestVM(1)
	vmID1 := vm.GetId()
	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	nonExistentVMID := uuid.NewTestUUID(9).String()

	// Test existing VM
	exists, err := s.datastore.Exists(s.sacCtx, vmID1)
	s.NoError(err)
	s.True(exists)

	// Test non-existing VM
	exists, err = s.datastore.Exists(s.sacCtx, nonExistentVMID)
	s.NoError(err)
	s.False(exists)

	// Test empty ID
	exists, err = s.datastore.Exists(s.sacCtx, "")
	s.Error(err)
	s.False(exists)
	s.Contains(err.Error(), "ERROR: invalid input syntax for type uuid: \"\"")

	// Test SAC denied:
	// Namespace-scoped resource, therefore access attempt to not authorized object
	// does not return error and behaves as if the object did not exist.
	exists, err = s.datastore.Exists(s.noSacCtx, vmID1)
	s.NoError(err)
	s.False(exists)
}

// TODO: Actually test concurrent writes
// Test concurrent reads with writes
func (s *VirtualMachineDataStoreTestSuite) TestConcurrentReads() {
	testVMCount := 10
	vmIDs := make([]string, 0, testVMCount)
	// Pre-populate some VMs
	for i := 1; i <= testVMCount; i++ {
		vm := s.createTestVM(i)
		vmIDs = append(vmIDs, vm.GetId())
		err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
		s.NoError(err)
	}

	numReaders := 5
	numReads := 50

	var wg sync.WaitGroup
	wg.Add(numReaders)

	errors := make(chan error, numReaders*numReads)

	// Launch multiple goroutines performing concurrent reads
	for i := range numReaders {
		go func(routineID int) {
			defer wg.Done()

			for j := range numReads {
				vmID := vmIDs[j%testVMCount]

				// Test GetVirtualMachine
				if _, _, err := s.datastore.GetVirtualMachine(s.sacCtx, vmID); err != nil {
					errors <- err
					continue
				}

				// Test Exists
				if _, err := s.datastore.Exists(s.sacCtx, vmID); err != nil {
					errors <- err
					continue
				}

				// Test CountVirtualMachines
				if _, err := s.datastore.CountVirtualMachines(s.sacCtx); err != nil {
					errors <- err
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		s.Fail("Concurrent read operation failed: %v", err)
	}
}

// Test data cloning
func (s *VirtualMachineDataStoreTestSuite) TestDataCloning() {
	vm := s.createTestVM(1)
	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Get VM
	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vm.GetId())
	s.NoError(err)
	s.True(found)

	// Modify retrieved VM
	originalName := retrievedVM.GetName()
	retrievedVM.Name = "modified-name"

	// Get VM again to verify original wasn't modified
	retrievedVM2, found, err := s.datastore.GetVirtualMachine(s.sacCtx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(originalName, retrievedVM2.GetName())
	s.NotEqual("modified-name", retrievedVM2.GetName())
}

// Test error handling edge cases
func (s *VirtualMachineDataStoreTestSuite) TestErrorHandling() {
	nonExistentVMID := uuid.NewTestUUID(9).String()
	// Test that GetVirtualMachine returns nil for non-existent VM, not empty struct
	vm, found, err := s.datastore.GetVirtualMachine(s.sacCtx, nonExistentVMID)
	s.NoError(err)
	s.False(found)
	s.Nil(vm)

	// Test empty ID validation
	vm, found, err = s.datastore.GetVirtualMachine(s.sacCtx, "")
	s.Error(err)
	s.False(found)
	s.Nil(vm)

	// Test Exists with empty ID
	exists, err := s.datastore.Exists(s.sacCtx, "")
	s.Error(err)
	s.False(exists)
	s.Contains(err.Error(), "ERROR: invalid input syntax for type uuid: \"\"")

	// Test Upsert with empty ID
	vmNoId := &storage.VirtualMachine{Name: "test-vm-no-id"}
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vmNoId)
	s.Error(err)
	s.Contains(err.Error(), "cannot upsert a virtualMachine without an id")
}

// Test GetAllVirtualMachines slice initialization
func (s *VirtualMachineDataStoreTestSuite) TestGetAllVirtualMachinesSliceHandling() {
	// Test with empty datastore
	vms, err := s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Empty(vms)
	s.Equal(0, len(vms))
	s.Equal(10, cap(vms))

	testVMCount := 3
	// Add some VMs
	for i := 1; i <= testVMCount; i++ {
		vm := s.createTestVM(i)
		err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
		s.NoError(err)
	}

	// Test with populated datastore
	vms, err = s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Len(vms, 3)
	// Verify slice was properly initialized with capacity
	s.GreaterOrEqual(cap(vms), 3)
}

// Benchmark tests
func BenchmarkUpsertVirtualMachine(b *testing.B) {
	db := pgtest.ForT(b)
	store := vmStore.New(db)
	ds := newDatastoreImpl(store)
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := &storage.VirtualMachine{
			Id:   uuid.NewTestUUID(i).String(),
			Name: fmt.Sprintf("test-vm-%d", i),
		}
		if err := ds.UpsertVirtualMachine(ctx, vm); err != nil {
			b.Errorf("Failed to create virtual machine: %v", err)
		}
	}
}

func BenchmarkGetVirtualMachine(b *testing.B) {
	db := pgtest.ForT(b)
	store := vmStore.New(db)
	ds := newDatastoreImpl(store)
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))

	// Pre-populate
	for i := 0; i < 1000; i++ {
		vm := &storage.VirtualMachine{
			Id:   uuid.NewTestUUID(i).String(),
			Name: fmt.Sprintf("test-vm-%d", i),
		}
		if err := ds.UpsertVirtualMachine(ctx, vm); err != nil {
			b.Errorf("Failed to create virtual machine: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := ds.GetVirtualMachine(ctx, uuid.NewTestUUID(i%1000).String()); err != nil {
			b.Errorf("Failed to get virtual machine: %v", err)
		}
	}
}
