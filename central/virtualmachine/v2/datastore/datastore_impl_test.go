//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	pgStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestVirtualMachineV2DataStore(t *testing.T) {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(VirtualMachineV2DataStoreTestSuite))
}

type VirtualMachineV2DataStoreTestSuite struct {
	suite.Suite

	testDB    *pgtest.TestPostgres
	datastore DataStore
	ctx       context.Context
	noSacCtx  context.Context
}

func (s *VirtualMachineV2DataStoreTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *VirtualMachineV2DataStoreTestSuite) SetupTest() {
	_, err := s.testDB.Exec(s.ctx, "TRUNCATE virtual_machine_v2 CASCADE")
	s.Require().NoError(err)

	store := pgStore.New(s.testDB.DB, concurrency.NewKeyFence())
	s.datastore = newDatastoreImpl(store)
}

func (s *VirtualMachineV2DataStoreTestSuite) TearDownSuite() {
	s.testDB.Close()
}

func createTestVM(i int) *storage.VirtualMachineV2 {
	return &storage.VirtualMachineV2{
		Id:          uuid.NewTestUUID(i).String(),
		Name:        fmt.Sprintf("test-vm-%d", i),
		Namespace:   "default",
		ClusterId:   uuid.NewV5FromNonUUIDs("cluster", fmt.Sprintf("%d", i)).String(),
		ClusterName: "test-cluster",
		GuestOs:     "rhel9",
		State:       storage.VirtualMachineV2_RUNNING,
		Notes:       []storage.VirtualMachineV2_Note{storage.VirtualMachineV2_MISSING_SCAN_DATA},
		VsockCid:    int32(42 + i),
	}
}

func createTestScanParts(vmID string) common.VMScanParts {
	scanID := uuid.NewV4().String()
	compID := uuid.NewV4().String()
	cveID := uuid.NewV4().String()

	return common.VMScanParts{
		Scan: &storage.VirtualMachineScanV2{
			Id:     scanID,
			VmV2Id: vmID,
			ScanOs: "rhel9",
		},
		Components: []*storage.VirtualMachineComponentV2{
			{
				Id:              compID,
				VmScanId:        scanID,
				Name:            "openssl",
				Version:         "1.1.1",
				Source:          storage.SourceType_OS,
				OperatingSystem: "rhel:9",
			},
		},
		CVEs: []*storage.VirtualMachineCVEV2{
			{
				Id:            cveID,
				VmV2Id:        vmID,
				VmComponentId: compID,
				CveBaseInfo: &storage.CVEInfo{
					Cve:     "CVE-2024-0001",
					Summary: "test vulnerability",
				},
				PreferredCvss: 7.5,
				Severity:      storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				IsFixable:     true,
				HasFixedBy:    &storage.VirtualMachineCVEV2_FixedBy{FixedBy: "1.1.2"},
			},
		},
	}
}

// region Count tests

func (s *VirtualMachineV2DataStoreTestSuite) TestCountVirtualMachines() {
	// Nil query returns zero for empty store.
	count, err := s.datastore.CountVirtualMachines(s.ctx, nil)
	s.NoError(err)
	s.Equal(0, count)

	// Empty query returns zero for empty store.
	count, err = s.datastore.CountVirtualMachines(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// After upsert, count reflects the new VM.
	vm := createTestVM(1)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	count, err = s.datastore.CountVirtualMachines(s.ctx, nil)
	s.NoError(err)
	s.Equal(1, count)

	// SAC denied returns 0.
	count, err = s.datastore.CountVirtualMachines(s.noSacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)
}

// endregion Count tests

// region Get tests

func (s *VirtualMachineV2DataStoreTestSuite) TestGetVirtualMachine() {
	vm := createTestVM(1)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	// Get existing VM.
	got, found, err := s.datastore.GetVirtualMachine(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(vm.GetName(), got.GetName())

	// Get missing VM.
	_, found, err = s.datastore.GetVirtualMachine(s.ctx, uuid.NewTestUUID(99).String())
	s.NoError(err)
	s.False(found)

	// SAC denied.
	_, found, err = s.datastore.GetVirtualMachine(s.noSacCtx, vm.GetId())
	s.NoError(err)
	s.False(found)
}

func (s *VirtualMachineV2DataStoreTestSuite) TestGetManyVirtualMachines() {
	vm1 := createTestVM(1)
	vm2 := createTestVM(2)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm1))
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm2))

	nonExistentID := uuid.NewTestUUID(99).String()

	// Batch get returns found VMs and missing indices.
	results, missing, err := s.datastore.GetManyVirtualMachines(s.ctx, []string{vm1.GetId(), nonExistentID, vm2.GetId()})
	s.NoError(err)
	s.Len(results, 2)
	s.Equal([]int{1}, missing)

	// SAC denied returns nothing.
	results, _, err = s.datastore.GetManyVirtualMachines(s.noSacCtx, []string{vm1.GetId()})
	s.NoError(err)
	s.Empty(results)
}

// endregion Get tests

// region Upsert tests

func (s *VirtualMachineV2DataStoreTestSuite) TestUpsertVirtualMachine() {
	vm := createTestVM(1)

	// Upsert new VM.
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	got, found, err := s.datastore.GetVirtualMachine(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(vm.GetName(), got.GetName())

	// Upsert update.
	vm.Name = "updated-name"
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	got, found, err = s.datastore.GetVirtualMachine(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal("updated-name", got.GetName())

	// Empty ID rejected.
	err = s.datastore.UpsertVirtualMachine(s.ctx, &storage.VirtualMachineV2{Name: "no-id"})
	s.Error(err)
	s.Contains(err.Error(), "cannot upsert a virtual machine without an id")
}

func (s *VirtualMachineV2DataStoreTestSuite) TestUpsertScan() {
	vm := createTestVM(1)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	parts := createTestScanParts(vm.GetId())
	s.NoError(s.datastore.UpsertScan(s.ctx, vm.GetId(), parts))

	// Verify VM still accessible after scan upsert.
	got, found, err := s.datastore.GetVirtualMachine(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(vm.GetName(), got.GetName())

	// Empty VM ID rejected.
	err = s.datastore.UpsertScan(s.ctx, "", parts)
	s.Error(err)
	s.Contains(err.Error(), "cannot upsert scan without a VM id")
}

// endregion Upsert tests

// region Delete tests

func (s *VirtualMachineV2DataStoreTestSuite) TestDeleteVirtualMachines() {
	vm1 := createTestVM(1)
	vm2 := createTestVM(2)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm1))
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm2))

	// Delete existing VM.
	s.NoError(s.datastore.DeleteVirtualMachines(s.ctx, vm1.GetId()))

	_, found, err := s.datastore.GetVirtualMachine(s.ctx, vm1.GetId())
	s.NoError(err)
	s.False(found)

	// Delete missing VM does not error.
	s.NoError(s.datastore.DeleteVirtualMachines(s.ctx, uuid.NewTestUUID(99).String()))

	// Delete with cascade: VM with scan data.
	parts := createTestScanParts(vm2.GetId())
	s.NoError(s.datastore.UpsertScan(s.ctx, vm2.GetId(), parts))

	s.NoError(s.datastore.DeleteVirtualMachines(s.ctx, vm2.GetId()))

	_, found, err = s.datastore.GetVirtualMachine(s.ctx, vm2.GetId())
	s.NoError(err)
	s.False(found)
}

// endregion Delete tests

// region Search tests

func (s *VirtualMachineV2DataStoreTestSuite) TestSearch() {
	vm := createTestVM(1)
	s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))

	results, err := s.datastore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	// Query filtering.
	results, err = s.datastore.Search(s.ctx, search.NewQueryBuilder().
		AddExactMatches(search.VirtualMachineName, "nonexistent").ProtoQuery())
	s.NoError(err)
	s.Empty(results)
}

func (s *VirtualMachineV2DataStoreTestSuite) TestSearchRawVirtualMachines() {
	// Insert VMs with distinct names to verify default sort order.
	for i := 1; i <= 3; i++ {
		vm := createTestVM(i)
		vm.Namespace = fmt.Sprintf("ns-%d", 4-i) // reverse namespace order
		s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))
	}

	// Default sort: by name, then namespace.
	vms, err := s.datastore.SearchRawVirtualMachines(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(vms, 3)

	// Verify name ordering (test-vm-1, test-vm-2, test-vm-3).
	for i, vm := range vms {
		s.Equal(fmt.Sprintf("test-vm-%d", i+1), vm.GetName())
	}

	// Pagination: limit to 2.
	query := search.EmptyQuery()
	query.Pagination = &v1.QueryPagination{Limit: 2}
	vms, err = s.datastore.SearchRawVirtualMachines(s.ctx, query)
	s.NoError(err)
	s.Len(vms, 2)
}

// endregion Search tests

// region Walk tests

func (s *VirtualMachineV2DataStoreTestSuite) TestWalk() {
	for i := 1; i <= 3; i++ {
		vm := createTestVM(i)
		s.NoError(s.datastore.UpsertVirtualMachine(s.ctx, vm))
	}

	var walked []*storage.VirtualMachineV2
	err := s.datastore.Walk(s.ctx, func(vm *storage.VirtualMachineV2) error {
		walked = append(walked, vm)
		return nil
	})
	s.NoError(err)
	s.Len(walked, 3)
}

// endregion Walk tests
