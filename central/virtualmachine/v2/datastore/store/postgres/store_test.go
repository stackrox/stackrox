//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type VMStoreTestSuite struct {
	suite.Suite
	store  *storeImpl
	testDB *pgtest.TestPostgres
	ctx    context.Context
}

func TestVMStore(t *testing.T) {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(VMStoreTestSuite))
}

func (s *VMStoreTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *VMStoreTestSuite) SetupTest() {
	_, err := s.testDB.Exec(s.ctx, "TRUNCATE virtual_machine_v2 CASCADE")
	s.Require().NoError(err)
	s.store = New(s.testDB.DB, concurrency.NewKeyFence()).(*storeImpl)
}

func (s *VMStoreTestSuite) newVM() *storage.VirtualMachineV2 {
	return &storage.VirtualMachineV2{
		Id:          uuid.NewV4().String(),
		Name:        "test-vm",
		Namespace:   "default",
		ClusterId:   uuid.NewV4().String(),
		ClusterName: "test-cluster",
		GuestOs:     "rhel9",
		State:       storage.VirtualMachineV2_RUNNING,
		Notes:       []storage.VirtualMachineV2_Note{storage.VirtualMachineV2_MISSING_SCAN_DATA},
		VsockCid:    42,
	}
}

func (s *VMStoreTestSuite) newScanParts(vmID string) common.VMScanParts {
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
		SourceComponents: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "openssl",
				Version: "1.1.1",
				Source:  storage.SourceType_OS,
				Vulnerabilities: []*storage.VirtualMachineVulnerability{
					{
						CveBaseInfo: &storage.VirtualMachineCVEInfo{
							Cve:     "CVE-2024-0001",
							Summary: "test vulnerability",
						},
						Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						SetFixedBy: &storage.VirtualMachineVulnerability_FixedBy{FixedBy: "1.1.2"},
						Cvss:       7.5,
					},
				},
			},
		},
	}
}

// region UpsertVM tests

func (s *VMStoreTestSuite) TestUpsertVM_New() {
	vm := s.newVM()

	s.NoError(s.store.UpsertVM(s.ctx, vm))

	got, exists, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(vm.GetName(), got.GetName())
	s.NotNil(got.GetLastUpdated())
	s.NotZero(got.GetHash())
}

func (s *VMStoreTestSuite) TestUpsertVM_Unchanged() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	firstGet, _, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	firstUpdated := firstGet.GetLastUpdated().AsTime()

	// Sleep to ensure timestamp difference.
	time.Sleep(10 * time.Millisecond)

	// Upsert again with same data (no field changes).
	vmCopy := vm.CloneVT()
	vmCopy.LastUpdated = nil
	vmCopy.Hash = 0
	s.NoError(s.store.UpsertVM(s.ctx, vmCopy))

	secondGet, _, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(secondGet.GetLastUpdated().AsTime().After(firstUpdated), "timestamp should be updated")
	// Hash should remain the same since data didn't change.
	s.Equal(firstGet.GetHash(), secondGet.GetHash())
}

func (s *VMStoreTestSuite) TestUpsertVM_Changed() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	firstGet, _, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)

	// Change a field.
	vm.GuestOs = "ubuntu22"
	vm.LastUpdated = nil
	vm.Hash = 0
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	secondGet, _, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	s.Equal("ubuntu22", secondGet.GetGuestOs())
	s.NotEqual(firstGet.GetHash(), secondGet.GetHash(), "hash should change when data changes")
}

// endregion UpsertVM tests

// region UpsertScan tests

func (s *VMStoreTestSuite) TestUpsertScan_New() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	// Verify scan was inserted.
	scan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	s.NotNil(scan)
	s.Equal(vm.GetId(), scan.GetVmV2Id())
	s.NotZero(scan.GetHash())

	// Verify components.
	components, err := s.getComponentsForScan(scan.GetId())
	s.NoError(err)
	s.Len(components, 1)
	s.Equal("openssl", components[0].GetName())

	// Verify CVEs.
	cves, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Len(cves, 1)
	s.Equal("CVE-2024-0001", cves[0].GetCveBaseInfo().GetCve())
}

func (s *VMStoreTestSuite) TestUpsertScan_Unchanged() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	firstScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	firstScanTime := firstScan.GetScanTime().AsTime()

	time.Sleep(10 * time.Millisecond)

	// Upsert with same components and CVEs (new scan ID but same content).
	// SourceComponents are identical since content hasn't changed.
	parts2 := s.newScanParts(vm.GetId())

	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts2))

	secondScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	// Scan time should be updated.
	s.True(secondScan.GetScanTime().AsTime().After(firstScanTime), "scan time should be updated")
	// Scan ID should remain the same (timestamp-only update).
	s.Equal(firstScan.GetId(), secondScan.GetId())
}

func (s *VMStoreTestSuite) TestUpsertScan_Changed() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	firstScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)

	time.Sleep(10 * time.Millisecond)

	// Change scan data: add a second component and CVE.
	parts2 := s.newScanParts(vm.GetId())
	newCompID := uuid.NewV4().String()
	parts2.Components = append(parts2.Components, &storage.VirtualMachineComponentV2{
		Id:              newCompID,
		VmScanId:        parts2.Scan.GetId(),
		Name:            "curl",
		Version:         "7.68.0",
		Source:          storage.SourceType_OS,
		OperatingSystem: "rhel:9",
	})
	parts2.CVEs = append(parts2.CVEs, &storage.VirtualMachineCVEV2{
		Id:            uuid.NewV4().String(),
		VmV2Id:        vm.GetId(),
		VmComponentId: newCompID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:     "CVE-2024-0002",
			Summary: "another vulnerability",
		},
		PreferredCvss: 5.0,
		Severity:      storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
	})
	parts2.SourceComponents = append(parts2.SourceComponents, &storage.EmbeddedVirtualMachineScanComponent{
		Name:    "curl",
		Version: "7.68.0",
		Source:  storage.SourceType_OS,
		Vulnerabilities: []*storage.VirtualMachineVulnerability{
			{
				CveBaseInfo: &storage.VirtualMachineCVEInfo{
					Cve:     "CVE-2024-0002",
					Summary: "another vulnerability",
				},
				Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				Cvss:     5.0,
			},
		},
	})

	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts2))

	secondScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	// Hash should differ since content changed.
	s.NotEqual(firstScan.GetHash(), secondScan.GetHash())
	// Should be a new scan ID (full replace).
	s.NotEqual(firstScan.GetId(), secondScan.GetId())

	// Verify two components and two CVEs now.
	components, err := s.getComponentsForScan(secondScan.GetId())
	s.NoError(err)
	s.Len(components, 2)

	cves, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Len(cves, 2)
}

// endregion UpsertScan tests

// region CVE created_at preservation tests

func (s *VMStoreTestSuite) TestUpsertScan_CVECreatedAtPreservation() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	// First scan with a specific created_at.
	oldTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	parts := s.newScanParts(vm.GetId())
	parts.CVEs[0].CveBaseInfo.CreatedAt = timestamppb.New(oldTime)
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	// Get the stored CVE created_at.
	cves, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Require().Len(cves, 1)
	firstCreatedAt := cves[0].GetCveBaseInfo().GetCreatedAt().AsTime()

	time.Sleep(10 * time.Millisecond)

	// Second scan with same CVE but newer created_at.
	parts2 := s.newScanParts(vm.GetId())
	parts2.CVEs[0].CveBaseInfo.Cve = "CVE-2024-0001"         // Same CVE
	parts2.CVEs[0].CveBaseInfo.CreatedAt = timestamppb.Now() // Newer timestamp
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts2))

	// Verify created_at was preserved (oldest wins).
	cves2, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Require().Len(cves2, 1)
	s.Equal(firstCreatedAt.UTC(), cves2[0].GetCveBaseInfo().GetCreatedAt().AsTime().UTC(),
		"created_at should be preserved from the first scan")
}

func (s *VMStoreTestSuite) TestUpsertScan_CVENilCveBaseInfo() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	// Add a CVE with nil CveBaseInfo — should not panic.
	parts.CVEs = append(parts.CVEs, &storage.VirtualMachineCVEV2{
		Id:            uuid.NewV4().String(),
		VmV2Id:        vm.GetId(),
		VmComponentId: parts.Components[0].GetId(),
		CveBaseInfo:   nil,
		PreferredCvss: 3.0,
		Severity:      storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
	})

	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	cves, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Len(cves, 2)
}

func (s *VMStoreTestSuite) TestUpsertScan_ScanOsChangeTriggersFullReplace() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	parts.Scan.ScanOs = "rhel9"
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	firstScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)

	time.Sleep(10 * time.Millisecond)

	// Upsert again with same components/CVEs but different ScanOs.
	parts2 := s.newScanParts(vm.GetId())
	parts2.Scan.ScanOs = "ubuntu22"

	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts2))

	secondScan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	// ScanOs change should trigger full replace — new scan ID.
	s.NotEqual(firstScan.GetId(), secondScan.GetId(), "ScanOs change should trigger full scan replace")
	s.NotEqual(firstScan.GetHash(), secondScan.GetHash(), "hash should differ when ScanOs changes")
	s.Equal("ubuntu22", secondScan.GetScanOs())
}

// endregion CVE created_at preservation tests

// region Delete tests

func (s *VMStoreTestSuite) TestDelete_Cascade() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	parts := s.newScanParts(vm.GetId())
	s.NoError(s.store.UpsertScan(s.ctx, vm.GetId(), parts))

	// Delete the VM.
	s.NoError(s.store.Delete(s.ctx, vm.GetId()))

	// VM should be gone.
	_, exists, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	s.False(exists)

	// Scan should be cascade deleted.
	scan, err := s.getScanForVM(vm.GetId())
	s.NoError(err)
	s.Nil(scan)

	// CVEs should be cascade deleted.
	cves, err := s.getCVEsForVM(vm.GetId())
	s.NoError(err)
	s.Empty(cves)
}

func (s *VMStoreTestSuite) TestDeleteMany() {
	vm1 := s.newVM()
	vm2 := s.newVM()
	vm2.Name = "vm2"
	s.NoError(s.store.UpsertVM(s.ctx, vm1))
	s.NoError(s.store.UpsertVM(s.ctx, vm2))

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(2, count)

	s.NoError(s.store.DeleteMany(s.ctx, []string{vm1.GetId(), vm2.GetId()}))

	count, err = s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)
}

// endregion Delete tests

// region Read operation tests

func (s *VMStoreTestSuite) TestCount() {
	vm1 := s.newVM()
	vm2 := s.newVM()
	vm2.Name = "vm2"
	s.NoError(s.store.UpsertVM(s.ctx, vm1))
	s.NoError(s.store.UpsertVM(s.ctx, vm2))

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(2, count)
}

func (s *VMStoreTestSuite) TestSearch() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	results, err := s.store.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *VMStoreTestSuite) TestGetMany() {
	vm1 := s.newVM()
	vm2 := s.newVM()
	vm2.Name = "vm2"
	s.NoError(s.store.UpsertVM(s.ctx, vm1))
	s.NoError(s.store.UpsertVM(s.ctx, vm2))

	nonExistentID := uuid.NewV4().String()
	results, missing, err := s.store.GetMany(s.ctx, []string{vm1.GetId(), nonExistentID, vm2.GetId()})
	s.NoError(err)
	s.Len(results, 2)
	s.Equal([]int{1}, missing)

	protoassert.SlicesEqual(s.T(), []*storage.VirtualMachineV2{vm1, vm2}, results)
}

// region EnsureVMExists tests

func (s *VMStoreTestSuite) TestEnsureVMExists_New() {
	vmID := uuid.NewV4().String()
	clusterID := uuid.NewV4().String()

	s.NoError(s.store.EnsureVMExists(s.ctx, vmID, clusterID))

	got, exists, err := s.store.Get(s.ctx, vmID)
	s.NoError(err)
	s.True(exists)
	s.Equal(vmID, got.GetId())
	s.Equal(clusterID, got.GetClusterId())
}

func (s *VMStoreTestSuite) TestEnsureVMExists_DoesNotClobber() {
	vm := s.newVM()
	s.NoError(s.store.UpsertVM(s.ctx, vm))

	// EnsureVMExists with the same ID should not overwrite existing data.
	s.NoError(s.store.EnsureVMExists(s.ctx, vm.GetId(), vm.GetClusterId()))

	got, exists, err := s.store.Get(s.ctx, vm.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(vm.GetName(), got.GetName(), "EnsureVMExists should not clobber existing VM name")
	s.Equal(vm.GetGuestOs(), got.GetGuestOs(), "EnsureVMExists should not clobber existing GuestOs")
	s.Equal(vm.GetState(), got.GetState(), "EnsureVMExists should not clobber existing State")
}

func (s *VMStoreTestSuite) TestEnsureVMExists_InvalidIDs() {
	s.Error(s.store.EnsureVMExists(s.ctx, "", uuid.NewV4().String()))
	s.Error(s.store.EnsureVMExists(s.ctx, uuid.NewV4().String(), ""))
	s.Error(s.store.EnsureVMExists(s.ctx, "not-a-uuid", uuid.NewV4().String()))
}

// endregion EnsureVMExists tests

// endregion Read operation tests

// region Helpers

func (s *VMStoreTestSuite) getScanForVM(vmID string) (*storage.VirtualMachineScanV2, error) {
	tx, ctx, err := s.store.begin(s.ctx)
	if err != nil {
		return nil, err
	}
	defer postgres.FinishReadOnlyTransaction(tx)
	return s.store.getScanForVM(ctx, tx, vmID)
}

func (s *VMStoreTestSuite) getComponentsForScan(scanID string) ([]*storage.VirtualMachineComponentV2, error) {
	rows, err := s.testDB.DB.Query(s.ctx, "SELECT serialized FROM "+componentTable+" WHERE vmscanid = $1", pgutils.NilOrUUID(scanID))
	if err != nil {
		return nil, err
	}
	return pgutils.ScanRows[storage.VirtualMachineComponentV2, *storage.VirtualMachineComponentV2](rows)
}

func (s *VMStoreTestSuite) getCVEsForVM(vmID string) ([]*storage.VirtualMachineCVEV2, error) {
	rows, err := s.testDB.DB.Query(s.ctx, "SELECT serialized FROM "+cveTable+" WHERE vmv2id = $1", pgutils.NilOrUUID(vmID))
	if err != nil {
		return nil, err
	}
	return pgutils.ScanRows[storage.VirtualMachineCVEV2, *storage.VirtualMachineCVEV2](rows)
}

// endregion Helpers
