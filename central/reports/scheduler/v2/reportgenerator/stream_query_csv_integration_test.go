//go:build sql_integration

package reportgenerator

import (
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

// csvColumnIndex mirrors csvHeader positions from csv_gen.go so assertions
// below are self-documenting rather than using bare magic numbers.
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

// buildStreamQuery combines a deployments query from a report snapshot with a
// CVE fields filter, attaches the pagination / selects for the given image type,
// and returns the final query together with its schema.
func (s *NewDataModelEnhancedReportingTestSuite) buildStreamQuery(
	snap *storage.ReportSnapshot,
	collection *storage.ResourceCollection,
	parts *ReportQueryParts,
) (*v1.Query, *walker.Schema) {
	s.T().Helper()

	rQuery, err := s.reportGenerator.buildReportQuery(s.ctx, snap, collection, time.Time{})
	s.Require().NoError(err)

	cveFilterQ, err := search.ParseQuery(rQuery.CveFieldsQuery, search.MatchAllIfEmpty())
	s.Require().NoError(err)

	q := search.ConjunctionQuery(rQuery.DeploymentsQuery, cveFilterQ)
	q.Pagination = parts.Pagination
	q.Selects = parts.Selects
	return q, parts.Schema
}

// runStream runs streamQueryToCSV and returns the parsed CSV rows plus the
// final row count.  refLinksCache is shared across calls when provided; pass
// nil to start with an empty cache.
func (s *NewDataModelEnhancedReportingTestSuite) runStream(
	schema *walker.Schema,
	query *v1.Query,
	refLinksCache map[string]string,
) (rows [][]string, rowCount int) {
	s.T().Helper()

	if refLinksCache == nil {
		refLinksCache = map[string]string{}
	}

	var buf bytes.Buffer
	csvW := csv.NewWriter(&buf)
	csvW.UseCRLF = true

	s.Require().NoError(
		s.reportGenerator.streamQueryToCSV(s.ctx, schema, query, csvW, refLinksCache, &rowCount),
	)
	csvW.Flush()
	s.Require().NoError(csvW.Error())

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	s.Require().NoError(err)
	return rows, rowCount
}

// runStreamDeployed is a convenience wrapper for the deployed-images query path.
func (s *NewDataModelEnhancedReportingTestSuite) runStreamDeployed(
	snap *storage.ReportSnapshot,
	collection *storage.ResourceCollection,
) (rows [][]string, rowCount int) {
	q, schema := s.buildStreamQuery(snap, collection, deployedImagesQueryParts)
	return s.runStream(schema, q, nil)
}

// runStreamWatched builds a watched-images query from a list of image full
// names plus an optional CVE fields filter string (empty = match all).
func (s *NewDataModelEnhancedReportingTestSuite) runStreamWatched(
	watchedImageFullNames []string,
	cveFieldsQuery string,
) (rows [][]string, rowCount int) {
	s.T().Helper()

	cveFilterQ, err := search.ParseQuery(cveFieldsQuery, search.MatchAllIfEmpty())
	s.Require().NoError(err)

	q := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ImageName, watchedImageFullNames...).ProtoQuery(),
		cveFilterQ,
	)
	q.Pagination = watchedImagesQueryParts.Pagination
	q.Selects = watchedImagesQueryParts.Selects

	return s.runStream(watchedImagesQueryParts.Schema, q, nil)
}

// collectColumn returns all values in a given column index across the rows.
func collectColumn(rows [][]string, col int) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row[col])
	}
	return out
}

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

// TestStreamQueryToCSV_AllDeployedImages verifies that with no CVE filter every
// deployed image × CVE pair appears as a row in the CSV.
//
// Test data (from SetupSuite): 4 deployments across 2 clusters × 2 namespaces,
// each with 1 image and 2 CVEs (fixable_critical + nonFixable_low) → 8 rows.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_AllDeployedImages() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(8, rowCount)
	s.Len(rows, 8)

	s.ElementsMatch([]string{
		"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
		"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
		"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
		"CVE-fixable_critical-c2_ns2_dep0_img_comp", "CVE-nonFixable_low-c2_ns2_dep0_img_comp",
	}, collectColumn(rows, colCVE))
}

// TestStreamQueryToCSV_FixableOnlyFilter verifies that Fixable=FIXABLE keeps
// only the fixable_critical CVEs.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_FixableOnlyFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_FIXABLE, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("true", row[colFixable], "row %d Fixable", i)
		s.Equal("CRITICAL", row[colSeverity], "row %d Severity", i)
		s.Contains(row[colCVE], "fixable_critical", "row %d CVE name", i)
	}
}

// TestStreamQueryToCSV_NotFixableFilter verifies that NOT_FIXABLE keeps only
// non-fixable low CVEs.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_NotFixableFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_NOT_FIXABLE, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("false", row[colFixable], "row %d Fixable", i)
		s.Equal("LOW", row[colSeverity], "row %d Severity", i)
		s.Contains(row[colCVE], "nonFixable_low", "row %d CVE name", i)
	}
}

// TestStreamQueryToCSV_CriticalSeverityFilter verifies that a CRITICAL-only
// severity filter returns only critical rows.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_CriticalSeverityFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH,
		[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("CRITICAL", row[colSeverity], "row %d Severity", i)
	}
}

// TestStreamQueryToCSV_LowSeverityFilter verifies that a LOW-only severity
// filter returns only low rows.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_LowSeverityFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH,
		[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY},
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("LOW", row[colSeverity], "row %d Severity", i)
		s.Equal("false", row[colFixable], "row %d Fixable", i)
	}
}

// TestStreamQueryToCSV_ClusterScopeFilter verifies scoping to cluster c1
// returns only c1 CVEs (4 rows: 2 deployments × 2 CVEs).
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_ClusterScopeFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "c1", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("c1", row[colCluster], "row %d Cluster", i)
	}
}

// TestStreamQueryToCSV_NamespaceScopeFilter verifies scoping to ns1 returns
// only ns1 CVEs from both clusters (4 rows: 2 clusters × 1 dep × 2 CVEs).
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_NamespaceScopeFilter() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "", "ns1", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(4, rowCount)
	s.Len(rows, 4)
	for i, row := range rows {
		s.Equal("ns1", row[colNamespace], "row %d Namespace", i)
	}
}

// TestStreamQueryToCSV_SingleDeploymentScope verifies scoping to a single
// deployment returns exactly that deployment's 2 CVEs with correct metadata.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_SingleDeploymentScope() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "c1", "ns1", "c1_ns1_dep0")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(2, rowCount)
	s.Require().Len(rows, 2)

	for i, row := range rows {
		s.Equal("c1", row[colCluster], "row %d Cluster", i)
		s.Equal("ns1", row[colNamespace], "row %d Namespace", i)
		s.Equal("c1_ns1_dep0", row[colDeployment], "row %d Deployment", i)
		s.Equal("c1_ns1_dep0_img", row[colImage], "row %d Image", i)
		s.Equal("c1_ns1_dep0_img_comp", row[colComponent], "row %d Component", i)
	}
	s.ElementsMatch([]string{
		"CVE-fixable_critical-c1_ns1_dep0_img_comp",
		"CVE-nonFixable_low-c1_ns1_dep0_img_comp",
	}, collectColumn(rows, colCVE))
}

// TestStreamQueryToCSV_RowCountMatchesParsedRows verifies the rowCount output
// parameter equals the number of CSV rows across multiple filter combinations.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_RowCountMatchesParsedRows() {
	testCases := map[string]struct {
		fixability storage.VulnerabilityReportFilters_Fixability
		severities []storage.VulnerabilitySeverity
		wantRows   int
	}{
		"both fixabilities, all severities": {
			storage.VulnerabilityReportFilters_BOTH, allSeverities(), 8,
		},
		"fixable only": {
			storage.VulnerabilityReportFilters_FIXABLE, allSeverities(), 4,
		},
		"not fixable only": {
			storage.VulnerabilityReportFilters_NOT_FIXABLE, allSeverities(), 4,
		},
		"critical severity only": {
			storage.VulnerabilityReportFilters_BOTH,
			[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY}, 4,
		},
	}

	for name, tc := range testCases {
		s.T().Run(name, func(t *testing.T) {
			snap := testReportSnapshot("col1", tc.fixability, tc.severities,
				[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
			rows, rowCount := s.runStreamDeployed(snap, testCollection("col1", "", "", ""))
			s.Equal(tc.wantRows, rowCount, "rowCount parameter")
			s.Equal(tc.wantRows, len(rows), "parsed CSV row count")
		})
	}
}

// TestStreamQueryToCSV_CSVColumnsHaveCorrectValues spot-checks every CSV column
// for the known fixable_critical CVE in c1/ns1/dep0.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_CSVColumnsHaveCorrectValues() {
	snap := testReportSnapshot("col1",
		storage.VulnerabilityReportFilters_FIXABLE,
		[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "c1", "ns1", "c1_ns1_dep0")

	rows, _ := s.runStreamDeployed(snap, collection)

	s.Require().Len(rows, 1, "expected exactly one row")
	row := rows[0]

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
	s.Equal("8.50", row[colNVDCVSS])
	// EPSS probability 0.7 formatted as percentage with 3 decimal places.
	s.Equal("70.000", row[colEPSS])
	// Advisory fields set in testutils.go testImage().
	s.Equal("RHSA-2025-CVE-fixable", row[colAdvName])
	s.Equal("test-rhsa-link", row[colAdvLink])
	// Discovered At must be populated (not the "Not Available" fallback).
	s.NotEqual("Not Available", row[colDiscoveredAt])
}

// TestStreamQueryToCSV_EmptyResultForNoMatchingQuery verifies a query that
// matches no deployments produces zero rows and rowCount=0.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_EmptyResultForNoMatchingQuery() {
	snap := testReportSnapshot("col1", storage.VulnerabilityReportFilters_BOTH, allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)
	collection := testCollection("col1", "nonexistent-cluster", "", "")

	rows, rowCount := s.runStreamDeployed(snap, collection)

	s.Equal(0, rowCount)
	s.Empty(rows)
}

// TestStreamQueryToCSV_WatchedImagesAllCVEs verifies that watched images produce
// rows with empty Cluster, Namespace, and Deployment columns.
//
// Test data: 2 watched images (w0_img, w1_img) × 2 CVEs = 4 rows.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_WatchedImagesAllCVEs() {
	rows, rowCount := s.runStreamWatched([]string{"w0_img", "w1_img"}, "")

	s.Equal(4, rowCount)
	s.Len(rows, 4)

	for i, row := range rows {
		s.Empty(row[colCluster], "row %d Cluster (watched → empty)", i)
		s.Empty(row[colNamespace], "row %d Namespace (watched → empty)", i)
		s.Empty(row[colDeployment], "row %d Deployment (watched → empty)", i)
		s.NotEmpty(row[colImage], "row %d Image", i)
		s.NotEmpty(row[colCVE], "row %d CVE", i)
	}

	s.ElementsMatch([]string{
		"CVE-fixable_critical-w0_img_comp", "CVE-nonFixable_low-w0_img_comp",
		"CVE-fixable_critical-w1_img_comp", "CVE-nonFixable_low-w1_img_comp",
	}, collectColumn(rows, colCVE))
}

// TestStreamQueryToCSV_WatchedImagesFixableFilter verifies that Fixable:true on
// watched images returns only fixable CVEs.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_WatchedImagesFixableFilter() {
	rows, rowCount := s.runStreamWatched([]string{"w0_img", "w1_img"}, "Fixable:true")

	s.Equal(2, rowCount)
	s.Len(rows, 2)
	for i, row := range rows {
		s.Equal("true", row[colFixable], "row %d Fixable", i)
		s.Equal("CRITICAL", row[colSeverity], "row %d Severity", i)
	}
}

// TestStreamQueryToCSV_SharedRefLinkCacheAcrossQueries verifies that a shared
// refLinksCache is populated after the first call and reused by the second call;
// the total rowCount across both calls equals the sum of their individual counts.
func (s *NewDataModelEnhancedReportingTestSuite) TestStreamQueryToCSV_SharedRefLinkCacheAcrossQueries() {
	snap := testReportSnapshot("col1",
		storage.VulnerabilityReportFilters_FIXABLE,
		[]storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED}, nil)

	sharedCache := map[string]string{}

	// First call: c1 deployments.
	q1, schema := s.buildStreamQuery(snap, testCollection("col1", "c1", "", ""), deployedImagesQueryParts)
	_, count1 := s.runStream(schema, q1, sharedCache)

	cacheAfterFirst := len(sharedCache)
	s.Greater(cacheAfterFirst, 0, "cache should be populated after first call")

	// Second call: c2 deployments — cache already populated with the same CVE IDs.
	q2, _ := s.buildStreamQuery(snap, testCollection("col1", "c2", "", ""), deployedImagesQueryParts)
	_, count2 := s.runStream(schema, q2, sharedCache)

	// Cache size must not grow (same CVE IDs appear in c2 as in c1).
	s.Equal(cacheAfterFirst, len(sharedCache), "cache should not grow for already-seen CVE IDs")

	// c1 has 2 fixable_critical CVEs; c2 also has 2.
	s.Equal(2, count1)
	s.Equal(2, count2)
}
