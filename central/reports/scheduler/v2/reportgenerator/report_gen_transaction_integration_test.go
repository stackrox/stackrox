//go:build sql_integration

package reportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"time"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/uuid"
)

// CSV column indices matching csvHeader in csv_gen.go.
const (
	colCluster      = 0
	colNamespace    = 1
	colDeployment   = 2
	colImage        = 3
	colComponent    = 4
	colCompVersion  = 5
	colCVE          = 6
	colFixable      = 7
	colFixedBy      = 8
	colSeverity     = 9
	colCVSS         = 10
	colNVDCVSS      = 11
	colEPSS         = 12
	colDiscoveredAt = 13
	colReference    = 14
	colAdvName      = 15
	colAdvLink      = 16
)

// collectColumn extracts a single column from all rows.
func collectColumn(rows [][]string, col int) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row[col]
	}
	return out
}

// setupTransactionDeps creates a multi-connection pool and the datastores
// required by generateReportTransaction. The returned reportGeneratorImpl
// is a shallow copy of s.reportGenerator with db, blobStore, and
// reportSnapshotStore replaced.
func (s *NewDataModelEnhancedReportingTestSuite) setupTransactionDeps() (
	rg *reportGeneratorImpl,
	blobStore blobDS.Datastore,
	configStore reportConfigDS.DataStore,
) {
	s.T().Helper()

	// generateReportTransaction runs cursor reads, CVE lookups, and blob
	// store writes all within a single transaction (1 connection) on a
	// single goroutine, so no concurrent tx access occurs.
	poolCfg := s.testDB.DB.Config().Copy()
	poolCfg.Config.MaxConns = 1
	pool, err := postgres.New(context.Background(), poolCfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() { pool.Close() })

	blobStore = blobDS.NewTestDatastore(s.T(), pool)
	snapStore := reportSnapshotDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	configStore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	rgCopy := *s.reportGenerator
	rgCopy.db = pool
	rgCopy.blobStore = blobStore
	rgCopy.reportSnapshotStore = snapStore
	return &rgCopy, blobStore, configStore
}

// prepareSnapshot creates a report snapshot with a backing report configuration
// (FK constraint) and sets the notification method to DOWNLOAD so the streaming
// path is exercised.
func (s *NewDataModelEnhancedReportingTestSuite) prepareSnapshot(
	configStore reportConfigDS.DataStore,
	collectionID string,
	fixability storage.VulnerabilityReportFilters_Fixability,
	severities []storage.VulnerabilitySeverity,
	imageTypes []storage.VulnerabilityReportFilters_ImageType,
) *storage.ReportSnapshot {
	s.T().Helper()

	snap := testReportSnapshot(collectionID, fixability, severities, imageTypes, nil)
	configID := uuid.NewV4().String()
	snap.ReportConfigurationId = configID
	snap.ReportId = uuid.NewV4().String()
	snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD

	_, err := configStore.AddReportConfiguration(s.ctx, &storage.ReportConfiguration{
		Id:   configID,
		Name: "txn-test-config",
		Type: storage.ReportConfiguration_VULNERABILITY,
	})
	s.Require().NoError(err)
	return snap
}

// readBlobCSV fetches a zipped CSV from blob store, decompresses it, and returns
// the parsed CSV rows (including the header row).
func (s *NewDataModelEnhancedReportingTestSuite) readBlobCSV(
	blobStore blobDS.Datastore,
	configID, reportID string,
) [][]string {
	s.T().Helper()

	blobPath := common.GetReportBlobPath(configID, reportID)
	var buf bytes.Buffer
	_, found, err := blobStore.Get(reportGenCtx, blobPath, &buf)
	s.Require().NoError(err)
	s.Require().True(found, "blob not found at %s", blobPath)

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	s.Require().NoError(err)
	s.Require().Len(zipReader.File, 1, "expected exactly one entry in the ZIP archive")

	rc, err := zipReader.File[0].Open()
	s.Require().NoError(err)
	defer func() { _ = rc.Close() }()

	csvBytes, err := io.ReadAll(rc)
	s.Require().NoError(err)

	r := csv.NewReader(bytes.NewReader(csvBytes))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	s.Require().NoError(err)
	return rows
}

// ---------------------------------------------------------------------------
// Tests for generateReportTransaction
// ---------------------------------------------------------------------------

// TestGenerateReportTransaction_AllDeployedImages verifies the full pipeline:
// cursor reads in a single transaction → batched CSV → ZIP → blob store.
//
// Test data: 4 deployments × 2 CVEs = 8 rows, plus header.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_AllDeployedImages() {
	rg, blobStore, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "", "", ""),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)

	rows := s.readBlobCSV(blobStore, snap.GetReportConfigurationId(), snap.GetReportId())
	s.Require().Len(rows, 9, "1 header + 8 data rows")
	s.Equal(csvHeader, rows[0])

	dataRows := rows[1:]
	s.ElementsMatch([]string{
		"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
		"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
		"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
		"CVE-fixable_critical-c2_ns2_dep0_img_comp", "CVE-nonFixable_low-c2_ns2_dep0_img_comp",
	}, collectColumn(dataRows, colCVE))
}

// TestGenerateReportTransaction_FixableOnlyFilter verifies that the fixable
// filter is applied through the transactional pipeline.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_FixableOnlyFilter() {
	rg, blobStore, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_FIXABLE, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "", "", ""),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)

	rows := s.readBlobCSV(blobStore, snap.GetReportConfigurationId(), snap.GetReportId())
	s.Require().Len(rows, 5, "1 header + 4 fixable rows")

	for _, row := range rows[1:] {
		s.Equal("true", row[colFixable])
		s.Equal("CRITICAL", row[colSeverity])
	}
}

// TestGenerateReportTransaction_DeployedAndWatched verifies that both deployed
// and watched image queries are executed within the same transaction.
//
// Test data: 4 deployed CVEs + 4 watched CVEs (2 watched images × 2 CVEs) = 8 data rows.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_DeployedAndWatched() {
	rg, blobStore, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{
			storage.VulnerabilityReportFilters_DEPLOYED,
			storage.VulnerabilityReportFilters_WATCHED,
		})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "", "", ""),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)

	rows := s.readBlobCSV(blobStore, snap.GetReportConfigurationId(), snap.GetReportId())
	// 8 deployed (4 deps × 2 CVEs) + 4 watched (2 images × 2 CVEs) + 1 header = 13
	s.Require().Len(rows, 13, "1 header + 8 deployed + 4 watched")
}

// TestGenerateReportTransaction_EmptyResult verifies that a query matching
// no data produces a valid ZIP with a CSV containing only the header row.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_EmptyResult() {
	rg, blobStore, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "nonexistent-cluster", "", ""),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)

	rows := s.readBlobCSV(blobStore, snap.GetReportConfigurationId(), snap.GetReportId())
	s.Require().Len(rows, 1, "only header row for empty result")
	s.Equal(csvHeader, rows[0])
}

// TestGenerateReportTransaction_CSVFieldValues spot-checks individual CSV field
// values for a single known row to verify end-to-end correctness.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_CSVFieldValues() {
	rg, blobStore, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_FIXABLE,
		[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "c1", "ns1", "c1_ns1_dep0"),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)

	rows := s.readBlobCSV(blobStore, snap.GetReportConfigurationId(), snap.GetReportId())
	s.Require().Len(rows, 2, "1 header + 1 data row")

	row := rows[1]
	s.Equal("c1", row[colCluster])
	s.Equal("ns1", row[colNamespace])
	s.Equal("c1_ns1_dep0", row[colDeployment])
	s.Equal("c1_ns1_dep0_img", row[colImage])
	s.Equal("c1_ns1_dep0_img_comp", row[colComponent])
	s.Equal("1.0", row[colCompVersion])
	s.Equal("CVE-fixable_critical-c1_ns1_dep0_img_comp", row[colCVE])
	s.Equal("true", row[colFixable])
	s.Equal("1.1", row[colFixedBy])
	s.Equal("CRITICAL", row[colSeverity])
	s.Equal("9.00", row[colCVSS])
	s.Equal("10.00", row[colNVDCVSS])
	s.Equal("70.000", row[colEPSS])
	s.NotEqual("Not Available", row[colDiscoveredAt])
	s.Equal("RHSA-2025-CVE-fixable", row[colAdvName])
	s.Equal("test-rhsa-link", row[colAdvLink])
}

// TestGenerateReportTransaction_ReportStatusUpdated verifies that the report
// snapshot status is set to GENERATED after a successful run.
func (s *NewDataModelEnhancedReportingTestSuite) TestGenerateReportTransaction_ReportStatusUpdated() {
	rg, _, configStore := s.setupTransactionDeps()
	snap := s.prepareSnapshot(configStore, "col1",
		storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})

	err := rg.generateReportTransaction(s.ctx, &ReportRequest{
		ReportSnapshot: snap,
		Collection:     testCollection("col1", "", "", ""),
		DataStartTime:  time.Time{},
	})
	s.Require().NoError(err)
	s.Equal(storage.ReportStatus_GENERATED, snap.GetReportStatus().GetRunState())
	s.NotNil(snap.GetReportStatus().GetCompletedAt())
}
