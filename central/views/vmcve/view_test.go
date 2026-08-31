//go:build sql_integration

package vmcve

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/views/common"
	componentStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	cveStore "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/store/postgres"
	scanStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	vmV2Store "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	vmV2Postgres "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
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
	sharedCriticalCVE = "CVE-2024-0001"
	criticalCVE2      = "CVE-2024-0002"
	criticalCVE3      = "CVE-2024-0003"
	importantCVE      = "CVE-2024-0004"
	moderateCVE       = "CVE-2024-0005"
	lowCVE            = "CVE-2024-0006"
)

type severityWant struct {
	critical, criticalFixable   int
	important, importantFixable int
	moderate, moderateFixable   int
	low, lowFixable             int
}

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

	vmA     string
	vmB     string
	vmEmpty string
}

func (s *vmCVEViewTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	s.cveView = NewCVEView(s.db)
	s.vmPG = vmV2Postgres.New(s.db, concurrency.NewKeyFence())
	s.scanPG = scanStore.New(s.db)
	s.componentPG = componentStore.New(s.db)
	s.cvePG = cveStore.New(s.db)

	s.vmA = s.insertVM("vm-a")
	s.vmB = s.insertVM("vm-b")
	s.vmEmpty = s.insertVM("vm-empty")

	scanA := s.insertScan(s.vmA)
	opensslA := s.insertComponent(scanA, "openssl")
	glibcA := s.insertComponent(scanA, "glibc")
	s.insertCVE(s.vmA, opensslA, sharedCriticalCVE, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true)
	s.insertCVE(s.vmA, glibcA, sharedCriticalCVE, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true)
	s.insertCVE(s.vmA, glibcA, criticalCVE2, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, false)
	s.insertCVE(s.vmA, opensslA, criticalCVE3, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, false)
	s.insertCVE(s.vmA, opensslA, importantCVE, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, true)
	s.insertCVE(s.vmA, opensslA, moderateCVE, storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, false)
	s.insertCVE(s.vmA, glibcA, lowCVE, storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, true)

	scanB := s.insertScan(s.vmB)
	opensslB := s.insertComponent(scanB, "openssl")
	s.insertCVE(s.vmB, opensslB, sharedCriticalCVE, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true)
}

func (s *vmCVEViewTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *vmCVEViewTestSuite) TestCountCVEsBySeverity() {
	cases := map[string]struct {
		q    *v1.Query
		want severityWant
	}{
		"should count distinct CVEs on one VM, not components or VM IDs": {
			q: search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, s.vmA).ProtoQuery(),
			want: severityWant{
				critical: 3, criticalFixable: 1,
				important: 1, importantFixable: 1,
				moderate: 1,
				low:      1, lowFixable: 1,
			},
		},
		"should return zeros when the VM has no CVEs": {
			q:    search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, s.vmEmpty).ProtoQuery(),
			want: severityWant{},
		},
		"should count distinct CVEs across VMs, not affected VMs": {
			q: search.EmptyQuery(),
			want: severityWant{
				critical: 3, criticalFixable: 1,
				important: 1, importantFixable: 1,
				moderate: 1,
				low:      1, lowFixable: 1,
			},
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			got, err := s.cveView.CountCVEsBySeverity(s.ctx, tc.q)
			s.Require().NoError(err)
			s.assertSeverityCounts(got, tc.want)
		})
	}
}

func (s *vmCVEViewTestSuite) TestCountBySeverityPerVM() {
	rows, err := s.cveView.CountBySeverityPerVM(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)

	byVM := make(map[string]common.ResourceCountByCVESeverity, len(rows))
	for _, row := range rows {
		byVM[row.GetVMID()] = row.GetSeverityCounts()
	}

	s.Require().Contains(byVM, s.vmA)
	s.assertSeverityCounts(byVM[s.vmA], severityWant{
		critical: 3, criticalFixable: 1,
		important: 1, importantFixable: 1,
		moderate: 1,
		low:      1, lowFixable: 1,
	})

	s.Require().Contains(byVM, s.vmB)
	s.assertSeverityCounts(byVM[s.vmB], severityWant{
		critical: 1, criticalFixable: 1,
	})

	s.NotContains(byVM, s.vmEmpty)
}

func (s *vmCVEViewTestSuite) TestCountBySeverity() {
	cases := map[string]struct {
		q    *v1.Query
		want severityWant
	}{
		"should count VMs that have a CVE, not the CVEs themselves": {
			q: search.NewQueryBuilder().AddExactMatches(search.CVE, sharedCriticalCVE).ProtoQuery(),
			want: severityWant{
				critical: 2, criticalFixable: 2,
			},
		},
		"should count distinct VMs per severity across the fleet": {
			q: search.EmptyQuery(),
			want: severityWant{
				critical: 2, criticalFixable: 2,
				important: 1, importantFixable: 1,
				moderate: 1,
				low:      1, lowFixable: 1,
			},
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			got, err := s.cveView.CountBySeverity(s.ctx, tc.q)
			s.Require().NoError(err)
			s.assertSeverityCounts(got, tc.want)
		})
	}
}

func (s *vmCVEViewTestSuite) TestGet_countsVMsPerSeverity() {
	got, err := s.cveView.Get(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVE, sharedCriticalCVE).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(2, got[0].GetAffectedVMCount())
	s.assertSeverityCounts(got[0].GetVMsBySeverity(), severityWant{
		critical: 2, criticalFixable: 2,
	})
}

func (s *vmCVEViewTestSuite) assertSeverityCounts(got common.ResourceCountByCVESeverity, want severityWant) {
	s.Equal(want.critical, got.GetCriticalSeverityCount().GetTotal(), "critical total")
	s.Equal(want.criticalFixable, got.GetCriticalSeverityCount().GetFixable(), "critical fixable")
	s.Equal(want.important, got.GetImportantSeverityCount().GetTotal(), "important total")
	s.Equal(want.importantFixable, got.GetImportantSeverityCount().GetFixable(), "important fixable")
	s.Equal(want.moderate, got.GetModerateSeverityCount().GetTotal(), "moderate total")
	s.Equal(want.moderateFixable, got.GetModerateSeverityCount().GetFixable(), "moderate fixable")
	s.Equal(want.low, got.GetLowSeverityCount().GetTotal(), "low total")
	s.Equal(want.lowFixable, got.GetLowSeverityCount().GetFixable(), "low fixable")
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
