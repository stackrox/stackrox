package datastore

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestVirtualMachineDataStore(t *testing.T) {
	suite.Run(t, new(VirtualMachineDataStoreTestSuite))
}

type VirtualMachineDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore
	ctx       context.Context
	sacCtx    context.Context
	noSacCtx  context.Context
}

func (s *VirtualMachineDataStoreTestSuite) SetupTest() {
	s.datastore = newDatastoreImpl()
	s.ctx = context.Background()
	s.sacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *VirtualMachineDataStoreTestSuite) createTestVM(id string) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		Id:        id,
		Name:      "test-vm-" + id,
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
	vm := s.createTestVM("test-1")
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	count, err = s.datastore.CountVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Equal(1, count)

	// Test with SAC denied
	count, err = s.datastore.CountVirtualMachines(s.noSacCtx)
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
	s.Equal(0, count)
}

// Test GetVirtualMachine
func (s *VirtualMachineDataStoreTestSuite) TestGetVirtualMachine() {
	vm := s.createTestVM("test-1")
	err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Test successful get
	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "test-1")
	s.NoError(err)
	s.True(found)
	s.Equal(vm.GetId(), retrievedVM.GetId())
	s.Equal(vm.GetName(), retrievedVM.GetName())

	// Test not found
	retrievedVM, found, err = s.datastore.GetVirtualMachine(s.sacCtx, "non-existent")
	s.NoError(err)
	s.False(found)
	s.Nil(retrievedVM)

	// Test empty ID
	retrievedVM, found, err = s.datastore.GetVirtualMachine(s.sacCtx, "")
	s.Error(err)
	s.False(found)
	s.Nil(retrievedVM)
	s.Contains(err.Error(), "Please specify an id")

	// Test SAC denied
	retrievedVM, found, err = s.datastore.GetVirtualMachine(s.noSacCtx, "test-1")
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
	s.False(found)
}

// Test GetAllVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestGetAllVirtualMachines() {
	// Test empty store
	vms, err := s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Empty(vms)

	// Add multiple VMs
	vm1 := s.createTestVM("test-1")
	vm2 := s.createTestVM("test-2")

	err = s.datastore.CreateVirtualMachine(s.sacCtx, vm1)
	s.NoError(err)
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vm2)
	s.NoError(err)

	vms, err = s.datastore.GetAllVirtualMachines(s.sacCtx)
	s.NoError(err)
	s.Len(vms, 2)

	// Test SAC denied
	vms, err = s.datastore.GetAllVirtualMachines(s.noSacCtx)
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
	s.Empty(vms)
}

// Test CreateVirtualMachine
func (s *VirtualMachineDataStoreTestSuite) TestCreateVirtualMachine() {
	vm := s.createTestVM("test-1")

	// Test successful creation
	err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Test duplicate creation
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.Error(err)
	s.Contains(err.Error(), "Already exists")

	// Test empty ID
	vmNoId := &storage.VirtualMachine{Name: "test-vm-no-id"}
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vmNoId)
	s.Error(err)
	s.Contains(err.Error(), "cannot create a virtualMachine without an id")

	// Test SAC denied
	vm2 := s.createTestVM("test-2")
	err = s.datastore.CreateVirtualMachine(s.noSacCtx, vm2)
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
}

// Test UpsertVirtualMachine
func (s *VirtualMachineDataStoreTestSuite) TestUpsertVirtualMachine() {
	vm := s.createTestVM("test-1")

	// Test initial upsert (create)
	err := s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Test update
	vm.Name = "updated-name"
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "test-1")
	s.NoError(err)
	s.True(found)
	s.Equal("updated-name", retrievedVM.GetName())

	// Test empty ID
	vmNoId := &storage.VirtualMachine{Name: "test-vm-no-id"}
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vmNoId)
	s.Error(err)
	s.Contains(err.Error(), "cannot upsert a virtualMachine without an id")

	// Test SAC denied
	vm2 := s.createTestVM("test-2")
	err = s.datastore.UpsertVirtualMachine(s.noSacCtx, vm2)
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
}

// Test DeleteVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestDeleteVirtualMachines() {
	// Create test VMs
	vm1 := s.createTestVM("test-1")
	vm2 := s.createTestVM("test-2")

	err := s.datastore.CreateVirtualMachine(s.sacCtx, vm1)
	s.NoError(err)
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vm2)
	s.NoError(err)

	// Test successful deletion
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, "test-1")
	s.NoError(err)

	// Verify deletion
	_, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "test-1")
	s.NoError(err)
	s.False(found)

	// Test deletion of non-existent VM
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, "non-existent")
	s.Error(err)
	s.Contains(err.Error(), "not found")

	// Test batch deletion with some missing
	err = s.datastore.DeleteVirtualMachines(s.sacCtx, "test-2", "non-existent")
	s.Error(err)
	s.Contains(err.Error(), "not found")

	// Verify test-2 still exists (all-or-nothing behavior)
	_, found, err = s.datastore.GetVirtualMachine(s.sacCtx, "test-2")
	s.NoError(err)
	s.True(found)

	// Test SAC denied
	err = s.datastore.DeleteVirtualMachines(s.noSacCtx, "test-2")
	s.Error(err)
	s.Equal(sac.ErrResourceAccessDenied, err)
}

// Test Exists
func (s *VirtualMachineDataStoreTestSuite) TestExists() {
	vm := s.createTestVM("test-1")
	err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Test existing VM
	exists, err := s.datastore.Exists(s.sacCtx, "test-1")
	s.NoError(err)
	s.True(exists)

	// Test non-existing VM
	exists, err = s.datastore.Exists(s.sacCtx, "non-existent")
	s.NoError(err)
	s.False(exists)

	// Test empty ID
	exists, err = s.datastore.Exists(s.sacCtx, "")
	s.Error(err)
	s.False(exists)
	s.Contains(err.Error(), "Please specify a valid id")

	// Test SAC denied
	exists, err = s.datastore.Exists(s.noSacCtx, "test-1")
	s.Error(err)
	s.False(exists)
}

// Test concurrent reads with writes
func (s *VirtualMachineDataStoreTestSuite) TestConcurrentReads() {
	// Pre-populate some VMs
	for i := range 10 {
		vm := s.createTestVM(fmt.Sprintf("vm-%d", i))
		err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
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
				vmID := fmt.Sprintf("vm-%d", j%10)

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
	vm := s.createTestVM("test-1")
	err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	// Get VM
	retrievedVM, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "test-1")
	s.NoError(err)
	s.True(found)

	// Modify retrieved VM
	originalName := retrievedVM.GetName()
	retrievedVM.Name = "modified-name"

	// Get VM again to verify original wasn't modified
	retrievedVM2, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "test-1")
	s.NoError(err)
	s.True(found)
	s.Equal(originalName, retrievedVM2.GetName())
	s.NotEqual("modified-name", retrievedVM2.GetName())
}

// Test error handling edge cases
func (s *VirtualMachineDataStoreTestSuite) TestErrorHandling() {
	// Test that GetVirtualMachine returns nil for non-existent VM, not empty struct
	vm, found, err := s.datastore.GetVirtualMachine(s.sacCtx, "non-existent")
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
	s.Contains(err.Error(), "Please specify a valid id")

	// Test Create with empty ID
	vmNoId := &storage.VirtualMachine{Name: "test-vm-no-id"}
	err = s.datastore.CreateVirtualMachine(s.sacCtx, vmNoId)
	s.Error(err)
	s.Contains(err.Error(), "cannot create a virtualMachine without an id")

	// Test Upsert with empty ID
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
	s.Equal(0, cap(vms))

	// Add some VMs
	for i := range 3 {
		vm := s.createTestVM(fmt.Sprintf("vm-%d", i))
		err := s.datastore.CreateVirtualMachine(s.sacCtx, vm)
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
func BenchmarkCreateVirtualMachine(b *testing.B) {
	ds := newDatastoreImpl()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := &storage.VirtualMachine{
			Id:   fmt.Sprintf("vm-%d", i),
			Name: fmt.Sprintf("test-vm-%d", i),
		}
		ds.CreateVirtualMachine(ctx, vm)
	}
}

func BenchmarkGetVirtualMachine(b *testing.B) {
	ds := newDatastoreImpl()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))

	// Pre-populate
	for i := range 1000 {
		vm := &storage.VirtualMachine{
			Id:   fmt.Sprintf("vm-%d", i),
			Name: fmt.Sprintf("test-vm-%d", i),
		}
		ds.CreateVirtualMachine(ctx, vm)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ds.GetVirtualMachine(ctx, fmt.Sprintf("vm-%d", i%1000))
	}
}
