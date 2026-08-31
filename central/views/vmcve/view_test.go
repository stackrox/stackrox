//go:build sql_integration

package vmcve

import (
	"context"
	"testing"

	componentStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	cveStore "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/store/postgres"
	scanStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	vmV2Store "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	vmV2Postgres "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	sharedCVE = "CVE-2024-1001"
	lonelyCVE = "CVE-2024-1002"
)

func TestVMCVEView(t *testing.T) {
	t.Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "true")
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(vmCVEViewTestSuite))
}

type vmCVEViewTestSuite struct {
	suite.Suite

	db      *pgtest.TestPostgres
	cveView CveView
	ctx     context.Context

	vmPG        vmV2Store.Store
	scanPG      scanStore.Store
	componentPG componentStore.Store
	cvePG       cveStore.Store

	vmID string
}

func (s *vmCVEViewTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	s.cveView = NewCVEView(s.db)
	s.vmPG = vmV2Postgres.New(s.db, concurrency.NewKeyFence())
	s.scanPG = scanStore.New(s.db)
	s.componentPG = componentStore.New(s.db)
	s.cvePG = cveStore.New(s.db)

	s.vmID = s.insertVM("vm-a")
	scanID := s.insertScan(s.vmID)
	openssl := s.insertComponent(scanID, "openssl")
	glibc := s.insertComponent(scanID, "glibc")
	s.insertCVE(s.vmID, openssl, sharedCVE, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true)
	s.insertCVE(s.vmID, glibc, sharedCVE, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true)
	s.insertCVE(s.vmID, openssl, lonelyCVE, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, false)
}

func (s *vmCVEViewTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *vmCVEViewTestSuite) TestCount_distinctCVEsNotStorageRows() {
	q := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, s.vmID).ProtoQuery()
	got, err := s.cveView.Count(s.ctx, q)
	s.Require().NoError(err)
	s.Equal(2, got)
}

func (s *vmCVEViewTestSuite) TestGetCVEsByVM() {
	q := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, s.vmID).ProtoQuery()
	got, err := s.cveView.GetCVEsByVM(s.ctx, q)
	s.Require().NoError(err)
	s.Require().Len(got, 2)

	byCVE := make(map[string]CVEByVMCore, len(got))
	for _, row := range got {
		s.NotContains(byCVE, row.GetCVE(), "duplicate CVE %s", row.GetCVE())
		byCVE[row.GetCVE()] = row
	}

	s.Require().Contains(byCVE, sharedCVE)
	s.Equal(2, byCVE[sharedCVE].GetAffectedComponentCount())
	s.Equal(int32(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY), byCVE[sharedCVE].GetMaxSeverity())
	s.True(byCVE[sharedCVE].GetIsFixable())

	s.Require().Contains(byCVE, lonelyCVE)
	s.Equal(1, byCVE[lonelyCVE].GetAffectedComponentCount())
	s.Equal(int32(storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY), byCVE[lonelyCVE].GetMaxSeverity())
	s.False(byCVE[lonelyCVE].GetIsFixable())
}

func (s *vmCVEViewTestSuite) insertVM(name string) string {
	vmID := uuid.NewV5FromNonUUIDs(testconsts.Cluster1, name).String()
	s.Require().NoError(s.vmPG.UpsertVM(s.ctx, &storage.VirtualMachineV2{
		Id:          vmID,
		Name:        name,
		Namespace:   testconsts.NamespaceA,
		ClusterId:   testconsts.Cluster1,
		ClusterName: "cluster-1",
	}))
	return vmID
}

func (s *vmCVEViewTestSuite) insertScan(vmID string) string {
	scanID := uuid.NewV5FromNonUUIDs(vmID, "scan").String()
	s.Require().NoError(s.scanPG.Upsert(s.ctx, &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanOs:   "rhel:9",
		ScanTime: timestamppb.Now(),
	}))
	return scanID
}

func (s *vmCVEViewTestSuite) insertComponent(scanID, name string) string {
	componentID := uuid.NewV5FromNonUUIDs(scanID, name).String()
	s.Require().NoError(s.componentPG.Upsert(s.ctx, &storage.VirtualMachineComponentV2{
		Id:              componentID,
		VmScanId:        scanID,
		Name:            name,
		Version:         "1.0.0",
		Source:          storage.SourceType_OS,
		OperatingSystem: "rhel:9",
	}))
	return componentID
}

func (s *vmCVEViewTestSuite) insertCVE(vmID, componentID, cve string, severity storage.VulnerabilitySeverity, fixable bool) {
	rowID := uuid.NewV5FromNonUUIDs(componentID, cve).String()
	obj := &storage.VirtualMachineCVEV2{
		Id:            rowID,
		VmV2Id:        vmID,
		VmComponentId: componentID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:         cve,
			PublishedOn: timestamppb.Now(),
			CreatedAt:   timestamppb.Now(),
		},
		Severity:  severity,
		IsFixable: fixable,
	}
	if fixable {
		obj.HasFixedBy = &storage.VirtualMachineCVEV2_FixedBy{FixedBy: "1.0.1"}
	}
	s.Require().NoError(s.cvePG.Upsert(s.ctx, obj))
}
