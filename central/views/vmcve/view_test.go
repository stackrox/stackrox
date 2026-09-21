//go:build sql_integration

package vmcve

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/views/common"
	componentStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	cveStore "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/store/postgres"
	scanStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	vmStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	vmV2Postgres "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const splitCVEID = "CVE-2024-SPLIT"

func TestVMCVEView(t *testing.T) {
	t.Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "true")
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(vmCVEViewTestSuite))
}

type vmCVEViewTestSuite struct {
	suite.Suite

	db          *pgtest.TestPostgres
	cveView     CveView
	vmStore     vmStore.Store
	scanPG      scanStore.Store
	componentPG componentStore.Store
	cvePG       cveStore.Store
}

func (s *vmCVEViewTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	s.cveView = NewCVEView(s.db)
	s.vmStore = vmV2Postgres.New(s.db, concurrency.NewKeyFence())
	s.scanPG = scanStore.New(s.db)
	s.componentPG = componentStore.New(s.db)
	s.cvePG = cveStore.New(s.db)
}

func (s *vmCVEViewTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

// TestSplitSeverityCountedOnceAtMax covers one CVE on openssl (Critical) and
// glibc (Important): chips follow max severity, matching the CVE table.
func (s *vmCVEViewTestSuite) TestSplitSeverityCountedOnceAtMax() {
	ctx := sac.WithAllAccess(context.Background())
	vmID := s.insertSplitSeverityCVE(ctx)
	q := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, vmID).ProtoQuery()

	rows, err := s.cveView.CountBySeverityPerVM(ctx, q)
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.requireOneCriticalChip(rows[0].GetSeverityCounts())

	counts, err := s.cveView.CountBySeverity(ctx, q, search.CVE)
	s.Require().NoError(err)
	s.requireOneCriticalChip(counts)
}

func (s *vmCVEViewTestSuite) TestCountBySeverity_AffectedVMsUseMaxSeverity() {
	ctx := sac.WithAllAccess(context.Background())
	s.insertSplitSeverityCVE(ctx)
	s.insertImportantOnlyCVE(ctx)

	q := search.NewQueryBuilder().AddExactMatches(search.CVE, splitCVEID).ProtoQuery()
	counts, err := s.cveView.CountBySeverity(ctx, q, search.VirtualMachineID)
	s.Require().NoError(err)
	s.Equal(1, counts.GetCriticalSeverityCount().GetTotal())
	s.Equal(1, counts.GetImportantSeverityCount().GetTotal())
}

func (s *vmCVEViewTestSuite) requireOneCriticalChip(counts common.ResourceCountByCVESeverity) {
	s.T().Helper()
	s.Equal(1, counts.GetCriticalSeverityCount().GetTotal())
	s.Equal(1, counts.GetCriticalSeverityCount().GetFixable())
	s.Equal(0, counts.GetImportantSeverityCount().GetTotal())
	s.Equal(0, counts.GetModerateSeverityCount().GetTotal())
	s.Equal(0, counts.GetLowSeverityCount().GetTotal())
	s.Equal(0, counts.GetUnknownSeverityCount().GetTotal())
}

func (s *vmCVEViewTestSuite) insertSplitSeverityCVE(ctx context.Context) string {
	vmID, scanID := s.insertVMScan(ctx, "vm-split")
	opensslID := s.insertComponent(ctx, scanID, "openssl", "1.1.1")
	glibcID := s.insertComponent(ctx, scanID, "glibc", "2.34")
	s.Require().NoError(s.cvePG.Upsert(ctx, splitCVERow(vmID, opensslID, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, false)))
	s.Require().NoError(s.cvePG.Upsert(ctx, splitCVERow(vmID, glibcID, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, true)))
	return vmID
}

func (s *vmCVEViewTestSuite) insertImportantOnlyCVE(ctx context.Context) string {
	vmID, scanID := s.insertVMScan(ctx, "vm-important")
	compID := s.insertComponent(ctx, scanID, "glibc", "2.34")
	s.Require().NoError(s.cvePG.Upsert(ctx, splitCVERow(vmID, compID, storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, true)))
	return vmID
}

func (s *vmCVEViewTestSuite) insertVMScan(ctx context.Context, name string) (vmID, scanID string) {
	vmID = uuid.NewV4().String()
	s.Require().NoError(s.vmStore.UpsertVM(ctx, &storage.VirtualMachineV2{
		Id:        vmID,
		Name:      name,
		Namespace: "ns",
		ClusterId: uuid.NewV4().String(),
	}))
	scanID = uuid.NewV5FromNonUUIDs(vmID, "scan").String()
	s.Require().NoError(s.scanPG.Upsert(ctx, &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanTime: timestamppb.Now(),
	}))
	return vmID, scanID
}

func (s *vmCVEViewTestSuite) insertComponent(ctx context.Context, scanID, name, version string) string {
	id := uuid.NewV5FromNonUUIDs(scanID, name).String()
	s.Require().NoError(s.componentPG.Upsert(ctx, &storage.VirtualMachineComponentV2{
		Id:       id,
		VmScanId: scanID,
		Name:     name,
		Version:  version,
	}))
	return id
}

func splitCVERow(vmID, componentID string, sev storage.VulnerabilitySeverity, fixable bool) *storage.VirtualMachineCVEV2 {
	cve := &storage.VirtualMachineCVEV2{
		Id:            uuid.NewV5FromNonUUIDs(componentID, splitCVEID).String(),
		VmV2Id:        vmID,
		VmComponentId: componentID,
		CveBaseInfo:   &storage.CVEInfo{Cve: splitCVEID},
		Severity:      sev,
		IsFixable:     fixable,
	}
	if fixable {
		cve.HasFixedBy = &storage.VirtualMachineCVEV2_FixedBy{FixedBy: "2.35"}
	}
	return cve
}
