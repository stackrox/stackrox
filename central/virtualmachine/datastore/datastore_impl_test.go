//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"math"
	"testing"

	vmStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
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

// Test CountVirtualMachines
func (s *VirtualMachineDataStoreTestSuite) TestCountVirtualMachines() {
	// Test with SAC allowed
	count, err := s.datastore.CountVirtualMachines(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	// Add a VM
	vm := s.createTestVM(1)
	err = s.datastore.UpsertVirtualMachine(s.sacCtx, vm)
	s.NoError(err)

	count, err = s.datastore.CountVirtualMachines(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(1, count)

	count, err = s.datastore.CountVirtualMachines(s.sacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(1, count)

	// Test with SAC denied
	// Namespace-scoped resource silently returns 0 if access is denied
	count, err = s.datastore.CountVirtualMachines(s.noSacCtx, nil)
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

// Test UpdateVirtualMachineScan
func (s *VirtualMachineDataStoreTestSuite) TestUpdateVirtualMachineScan() {
	priorVirtualMachineScan := &storage.VirtualMachineScan{
		Notes: []storage.VirtualMachineScan_Note{
			storage.VirtualMachineScan_OS_UNKNOWN,
		},
	}
	testVirtualMachineScan := &storage.VirtualMachineScan{
		Components: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "my-test-package",
				Version: "1.2.3",
			},
		},
	}

	fullAccessCtx := sac.WithAllAccess(s.T().Context())
	// Inject a VM without scan data (to enrich)
	testVM1 := s.createTestVM(1)
	s.Require().NoError(s.datastore.UpsertVirtualMachine(fullAccessCtx, testVM1))
	expectedVM1 := testVM1.CloneVT()
	expectedVM1.Scan = testVirtualMachineScan
	// Inject a VM with scan data (to clean)
	testVM2 := s.createTestVM(2)
	testVM2.Scan = testVirtualMachineScan
	s.Require().NoError(s.datastore.UpsertVirtualMachine(fullAccessCtx, testVM2))
	expectedVM2 := testVM2.CloneVT()
	expectedVM2.Scan = nil
	// Inject a VM with scan data (to update)
	testVM3 := s.createTestVM(3)
	testVM3.Scan = priorVirtualMachineScan
	s.Require().NoError(s.datastore.UpsertVirtualMachine(fullAccessCtx, testVM3))
	expectedVM3 := testVM3.CloneVT()
	expectedVM3.Scan = testVirtualMachineScan

	tests := map[string]struct {
		targetVMID    string
		inputScan     *storage.VirtualMachineScan
		expectedError error
		expectedVM    *storage.VirtualMachine
	}{
		"Scan update to a non-existing VM returns NotFound error": {
			targetVMID:    uuid.NewTestUUID(0).String(),
			inputScan:     testVirtualMachineScan,
			expectedError: errox.NotFound,
		},
		"Scan update with scan to an existing VM with no scan results in VM with scan data": {
			targetVMID: testVM1.GetId(),
			inputScan:  testVirtualMachineScan,
			expectedVM: expectedVM1,
		},
		"Scan update with nil scan to an existing VM with scan data results in VM with no scan data": {
			targetVMID: testVM2.GetId(),
			inputScan:  nil,
			expectedVM: expectedVM2,
		},
		"Scan update with scan to an existing VM with prior scan data results in VM with updated scan data": {
			targetVMID: testVM3.GetId(),
			inputScan:  testVirtualMachineScan,
			expectedVM: expectedVM3,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			updateErr := s.datastore.UpdateVirtualMachineScan(fullAccessCtx, tc.targetVMID, tc.inputScan)
			if tc.expectedError != nil {
				s.ErrorIs(updateErr, tc.expectedError)
			} else {
				s.NoError(updateErr)

				updatedVM, found, fetchErr := s.datastore.GetVirtualMachine(fullAccessCtx, tc.targetVMID)
				s.NoError(fetchErr)
				s.True(found)
				protoassert.Equal(s.T(), tc.expectedVM, updatedVM)
			}
		})
	}
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
				if _, err := s.datastore.CountVirtualMachines(s.sacCtx, nil); err != nil {
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

func (s *VirtualMachineDataStoreTestSuite) TestSearchRawVirtualMachines() {
	s.populateDatabaseForSearch()
	testCases := map[string]struct {
		query                *v1.Query
		expectedDistribution map[string]map[string]int
	}{
		"Empty query retrieves the full dataset": {
			query: search.EmptyQuery(),
			expectedDistribution: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 2,
				},
				testconsts.Cluster2: {
					testconsts.NamespaceA: 2,
					testconsts.NamespaceC: 4,
				},
			},
		},
		"Query by cluster ID only retrieves information for the target cluster": {
			query: queryForClusterID(testconsts.Cluster1),
			expectedDistribution: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 2,
				},
			},
		},
		"Query by namespace only retrieves information for the target namespace name across clusters": {
			query: queryForNamespace(testconsts.NamespaceA),
			expectedDistribution: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
				},
				testconsts.Cluster2: {
					testconsts.NamespaceA: 2,
				},
			},
		},
		"Query by cluster ID and namespace retrieves information for the target namespace on the target cluster": {
			query: search.ConjunctionQuery(
				queryForClusterID(testconsts.Cluster2),
				queryForNamespace(testconsts.NamespaceA),
			),
			expectedDistribution: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 2,
				},
			},
		},
		"Query by VM ID only retrieves information for the target VM": {
			query: queryForObjectID(generateVirtualMachineID(testconsts.Cluster2, testconsts.NamespaceC, 2)),
			expectedDistribution: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceC: 1,
				},
			},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			ctx := sac.WithAllAccess(s.T().Context())
			virtualMachines, err := s.datastore.SearchRawVirtualMachines(ctx, tc.query)
			s.NoError(err)
			fetchedDistribution := countSearchResultsObjectsPerClusterAndNamespace(s.T(), virtualMachines)
			s.Equal(tc.expectedDistribution, fetchedDistribution)
		})
	}
}

func (s *VirtualMachineDataStoreTestSuite) TestSearchRawVirtualMachinesSort() {
	s.populateDatabaseForSearch()
	testCases := map[string]struct {
		query                *v1.Query
		expectedMachineIDs   []string
		expectedDistribution map[string]map[string]int
	}{
		"No sort option results in results sorted by name and namespace": {
			query: queryForClusterID(testconsts.Cluster1),
			expectedMachineIDs: []string{
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 1),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 1),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 2),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 2),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 3),
			},
		},
		"Sort by ID is done when requested": {
			query: addPagination(
				queryForClusterID(testconsts.Cluster1),
				sortBy(search.VirtualMachineID),
			),
			expectedMachineIDs: []string{
				// UUID starting with 09...
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 2),
				// UUID starting with 5b...
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 1),
				// UUID starting with 77...
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 2),
				// UUID starting with b4...
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 3),
				// UUID starting with fc...
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 1),
			},
		},
		"Custom sort options are applied (not the default)": {
			query: addPagination(
				queryForClusterID(testconsts.Cluster1),
				sortBy(search.Namespace, search.VirtualMachineName),
			),
			expectedMachineIDs: []string{
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 1),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 2),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceA, 3),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 1),
				generateVirtualMachineID(testconsts.Cluster1, testconsts.NamespaceB, 2),
			},
		},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			ctx := sac.WithAllAccess(s.T().Context())
			virtualMachines, err := s.datastore.SearchRawVirtualMachines(ctx, tc.query)
			s.NoError(err)
			vmIDs := make([]string, len(virtualMachines))
			for ix, vm := range virtualMachines {
				vmIDs[ix] = vm.GetId()
			}
			s.Equal(tc.expectedMachineIDs, vmIDs)
		})
	}
}

func (s *VirtualMachineDataStoreTestSuite) TestSearchRawVirtualMachinesSliceHandling() {
	for i := 0; i < 50; i++ {
		for _, testCluster := range []string{testconsts.Cluster1, testconsts.Cluster2} {
			for _, testNamespace := range []string{testconsts.NamespaceA, testconsts.NamespaceB, testconsts.NamespaceC} {
				err := s.injectNamespacedVirtualMachine(testCluster, testNamespace, i)
				s.Require().NoError(err)
			}
		}
	}

	for name, tc := range map[string]struct {
		query             *v1.Query
		expectedResultLen int
		expectedResultCap int
	}{
		"No limit on a narrow dataset should return results with default page size capacity": {
			query: search.ConjunctionQuery(
				queryForClusterID(testconsts.Cluster1),
				queryForNamespace(testconsts.NamespaceA),
			),
			expectedResultLen: 50,
			expectedResultCap: defaultPageSize,
		},
		"Negative limit on a narrow dataset should return results with default page size capacity": {
			query: addPagination(
				search.ConjunctionQuery(
					queryForClusterID(testconsts.Cluster1),
					queryForNamespace(testconsts.NamespaceA),
				),
				&v1.Pagination{Limit: -10},
			),
			expectedResultLen: 50,
			expectedResultCap: defaultPageSize,
		},
		"Limit lower than the default page size is used as result array capacity when provided": {
			query:             addPagination(search.EmptyQuery(), &v1.Pagination{Limit: 10}),
			expectedResultLen: 10,
			expectedResultCap: 10,
		},
		"Limit equal to the default page size is used as result array capacity when provided": {
			query:             addPagination(search.EmptyQuery(), &v1.Pagination{Limit: defaultPageSize}),
			expectedResultLen: defaultPageSize,
			expectedResultCap: defaultPageSize,
		},
		"Limit greater than the default page size is used as result array capacity when provided": {
			query:             addPagination(search.EmptyQuery(), &v1.Pagination{Limit: 2 * defaultPageSize}),
			expectedResultLen: 2 * defaultPageSize,
			expectedResultCap: 2 * defaultPageSize,
		},
	} {
		s.Run(name, func() {
			ctx := sac.WithAllAccess(s.T().Context())
			results, err := s.datastore.SearchRawVirtualMachines(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedResultLen)
			s.Equal(tc.expectedResultCap, cap(results))
		})
	}
}

func (s *VirtualMachineDataStoreTestSuite) TestSearchRawVirtualMachineSAC() {
	s.populateDatabaseForSearch()
	emptyDataSetDistribution := make(map[string]map[string]int)
	fullDataSetDistribution := map[string]map[string]int{
		testconsts.Cluster1: {
			testconsts.NamespaceA: 3,
			testconsts.NamespaceB: 2,
		},
		testconsts.Cluster2: {
			testconsts.NamespaceA: 2,
			testconsts.NamespaceC: 4,
		},
	}

	s.Run("Full access returns the whole data set", func() {
		ctx := sac.WithAllAccess(s.T().Context())
		results, err := s.datastore.SearchRawVirtualMachines(ctx, search.EmptyQuery())
		s.NoError(err)
		resultDistribution := countSearchResultsObjectsPerClusterAndNamespace(s.T(), results)
		s.Equal(fullDataSetDistribution, resultDistribution)
	})

	s.Run("Unauthorized access yields no results", func() {
		ctx := sac.WithNoAccess(s.T().Context())
		results, err := s.datastore.SearchRawVirtualMachines(ctx, search.EmptyQuery())
		s.NoError(err)
		resultDistribution := countSearchResultsObjectsPerClusterAndNamespace(s.T(), results)
		s.Equal(emptyDataSetDistribution, resultDistribution)
	})

	contexts := testutils.GetNamespaceScopedTestContexts(s.T().Context(), s.T(), resources.VirtualMachine)

	for name, tc := range map[string]testutils.SACSearchTestCase{
		"Full resource read-write access returns the full dataset": {
			ScopeKey: testutils.UnrestrictedReadWriteCtx,
			Results:  fullDataSetDistribution,
		},
		"Full resource read-only access returns the full dataset": {
			ScopeKey: testutils.UnrestrictedReadCtx,
			Results:  fullDataSetDistribution,
		},
		"Cluster-restricted access returns the data for the specified cluster (cluster1)": {
			ScopeKey: testutils.Cluster1ReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 2,
				},
			},
		},
		"Cluster-restricted access returns the data for the specified cluster (cluster2)": {
			ScopeKey: testutils.Cluster2ReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceA: 2,
					testconsts.NamespaceC: 4,
				},
			},
		},
		"Cluster-restricted access returns the data for the specified cluster (cluster3 -> none)": {
			ScopeKey: testutils.Cluster3ReadWriteCtx,
			Results:  emptyDataSetDistribution,
		},
		"Single cluster-namespace-restricted access returns the data for the specified scope (cluster 1, namespace A)": {
			ScopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
				},
			},
		},
		"Single cluster-namespace-restricted access returns the data for the specified scope (cluster 2, namespace C)": {
			ScopeKey: testutils.Cluster2NamespaceCReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster2: {
					testconsts.NamespaceC: 4,
				},
			},
		},
		"Access to multiple cluster-namespaces returns the data for the specified scope (cluster 1, namespaces AB)": {
			ScopeKey: testutils.Cluster1NamespacesABReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
					testconsts.NamespaceB: 2,
				},
			},
		},
		"Access to multiple cluster-namespaces returns the data for the specified scope (cluster 1, namespaces AC)": {
			ScopeKey: testutils.Cluster1NamespacesACReadWriteCtx,
			Results: map[string]map[string]int{
				testconsts.Cluster1: {
					testconsts.NamespaceA: 3,
				},
			},
		},
	} {
		s.Run(name, func() {
			ctx := contexts[tc.ScopeKey]
			results, err := s.datastore.SearchRawVirtualMachines(ctx, search.EmptyQuery())
			s.NoError(err)
			resultDistribution := countSearchResultsObjectsPerClusterAndNamespace(s.T(), results)
			s.Equal(tc.Results, resultDistribution)
		})
	}
	// basic SAC smoke tests
	// no access
	// full write only
	// full read only
	// full read-write ?
	// cluster read
	// cluster write
	// cluster-namespace read
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

// region helpers

func (s *VirtualMachineDataStoreTestSuite) createTestVM(id int) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		Id:        uuid.NewTestUUID(id).String(),
		Name:      fmt.Sprintf("test-vm-%d", id),
		Namespace: "default",
	}
}

func (s *VirtualMachineDataStoreTestSuite) populateDatabaseForSearch() {
	// 3 VMs in cluster 1 namespace A
	// 2 VMs in cluster 1 namespace B
	// 2 VMs in cluster 2 namespace A
	// 4 VMs in cluster 2 namespace C
	var err error
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster1, testconsts.NamespaceA, 1)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster1, testconsts.NamespaceA, 2)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster1, testconsts.NamespaceA, 3)
	s.Require().NoError(err)

	err = s.injectNamespacedVirtualMachine(testconsts.Cluster1, testconsts.NamespaceB, 1)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster1, testconsts.NamespaceB, 2)
	s.Require().NoError(err)

	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceA, 1)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceA, 2)
	s.Require().NoError(err)

	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceC, 1)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceC, 2)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceC, 3)
	s.Require().NoError(err)
	err = s.injectNamespacedVirtualMachine(testconsts.Cluster2, testconsts.NamespaceC, 4)
	s.Require().NoError(err)
}

func (s *VirtualMachineDataStoreTestSuite) injectNamespacedVirtualMachine(
	clusterID string,
	namespace string,
	index int,
) error {
	vm := createNamespacedTestVM(clusterID, namespace, index)
	ctx := sac.WithAllAccess(s.T().Context())
	return s.datastore.UpsertVirtualMachine(ctx, vm)
}

func createNamespacedTestVM(
	clusterID string,
	namespace string,
	index int,
) *storage.VirtualMachine {
	vmID := generateVirtualMachineID(clusterID, namespace, index)
	return &storage.VirtualMachine{
		Id:        vmID,
		Name:      fmt.Sprintf("Virtual Machine %d", index),
		ClusterId: clusterID,
		Namespace: namespace,
	}
}

func generateVirtualMachineID(
	clusterID string,
	namespace string,
	index int,
) string {
	return uuid.NewV5FromNonUUIDs(clusterID, fmt.Sprintf("%s-%d", namespace, index)).String()
}

func queryForObjectID(virtualMachineID string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_DocIdQuery{
					DocIdQuery: &v1.DocIDQuery{
						Ids: []string{virtualMachineID},
					},
				},
			},
		},
	}
}

func queryForClusterID(clusterID string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: search.ClusterID.String(),
						Value: clusterID,
					},
				},
			},
		},
	}
}

func queryForNamespace(ns string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: search.Namespace.String(),
						Value: ns,
					},
				},
			},
		},
	}
}

func addPagination(query *v1.Query, pagination *v1.Pagination) *v1.Query {
	returnedQuery := query.CloneVT()
	paginated.FillPagination(returnedQuery, pagination, math.MaxInt32)
	return returnedQuery
}

func sortBy(fields ...search.FieldLabel) *v1.Pagination {
	result := &v1.Pagination{}
	for _, field := range fields {
		result.SortOptions = append(result.SortOptions, &v1.SortOption{
			Field: field.String(),
		})
	}
	return result
}

func countSearchResultsObjectsPerClusterAndNamespace(t *testing.T, results []*storage.VirtualMachine) map[string]map[string]int {
	resultsAsNamespacedObjects := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, result := range results {
		resultsAsNamespacedObjects = append(resultsAsNamespacedObjects, result)
	}
	return testutils.CountSearchResultObjectsPerClusterAndNamespace(t, resultsAsNamespacedObjects)
}

// endregion helpers
