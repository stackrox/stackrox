package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExportCSV_Grouped(t *testing.T) {
	// Create test findings with duplicate CVEs but different advisories
	now := time.Now()
	findings := []*storage.ScanFinding{
		{
			Id:                    "finding-1",
			CveName:               "CVE-2024-0001",
			AdvisoryId:            "RHSA-2024:0001",
			SourceName:            "Red Hat",
			Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			Cvss:                  9.8,
			NvdCvss:               9.8,
			EpssPercentile:        0.95,
			IsFixable:             true,
			FixedBy:               "1.2.3",
			DataSource:            "nvd",
			Description:           "Critical vulnerability",
			ImageId:               "image-1",
			FirstSystemOccurrence: timestamppb.New(now),
		},
		{
			Id:                    "finding-2",
			CveName:               "CVE-2024-0001",
			AdvisoryId:            "GHSA-xxxx-yyyy-zzzz",
			SourceName:            "GitHub",
			Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:                  8.5,
			NvdCvss:               9.8,
			EpssPercentile:        0.95,
			IsFixable:             true,
			FixedBy:               "1.2.3",
			DataSource:            "nvd",
			Description:           "Critical vulnerability",
			ImageId:               "image-2",
			FirstSystemOccurrence: timestamppb.New(now.Add(-24 * time.Hour)),
		},
		{
			Id:                    "finding-3",
			CveName:               "CVE-2024-0002",
			AdvisoryId:            "RHSA-2024:0002",
			SourceName:            "Red Hat",
			Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			Cvss:                  5.5,
			NvdCvss:               5.5,
			EpssPercentile:        0.45,
			IsFixable:             false,
			FixedBy:               "",
			DataSource:            "nvd",
			Description:           "Moderate vulnerability",
			ImageId:               "image-1",
			FirstSystemOccurrence: timestamppb.New(now),
		},
	}

	var buf bytes.Buffer
	err := ExportCSV(&buf, findings, FormatGrouped)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 2 CVE rows
	require.Len(t, records, 3, "expected header + 2 CVE rows")

	// Verify header
	expectedHeader := []string{
		"CVE", "Severity", "CVSS", "Image Count", "Fixable",
		"Fixed In", "First Seen", "NVD CVSS", "EPSS", "Advisories",
	}
	assert.Equal(t, expectedHeader, records[0])

	// Find CVE-2024-0001 row
	var cve0001Row []string
	var cve0002Row []string
	for _, row := range records[1:] {
		if row[0] == "CVE-2024-0001" {
			cve0001Row = row
		} else if row[0] == "CVE-2024-0002" {
			cve0002Row = row
		}
	}

	require.NotNil(t, cve0001Row, "CVE-2024-0001 should be present")
	require.NotNil(t, cve0002Row, "CVE-2024-0002 should be present")

	// Verify CVE-2024-0001 (should use MAX severity = CRITICAL)
	assert.Equal(t, "CVE-2024-0001", cve0001Row[0])
	assert.Equal(t, "CRITICAL_VULNERABILITY_SEVERITY", cve0001Row[1])
	assert.Equal(t, "9.80", cve0001Row[2]) // MAX CVSS
	assert.Equal(t, "2", cve0001Row[3])    // 2 unique images
	assert.Equal(t, "true", cve0001Row[4])
	assert.Equal(t, "1.2.3", cve0001Row[5])
	assert.Equal(t, "9.80", cve0001Row[7]) // NVD CVSS
	assert.Equal(t, "0.95", cve0001Row[8]) // EPSS

	// Verify advisories are comma-separated
	advisories := cve0001Row[9]
	assert.True(t, strings.Contains(advisories, "RHSA-2024:0001"), "should contain RHSA advisory")
	assert.True(t, strings.Contains(advisories, "GHSA-xxxx-yyyy-zzzz"), "should contain GHSA advisory")
	assert.True(t, strings.Contains(advisories, ", "), "advisories should be comma-separated")

	// Verify CVE-2024-0002
	assert.Equal(t, "CVE-2024-0002", cve0002Row[0])
	assert.Equal(t, "MODERATE_VULNERABILITY_SEVERITY", cve0002Row[1])
	assert.Equal(t, "5.50", cve0002Row[2])
	assert.Equal(t, "1", cve0002Row[3]) // 1 image
	assert.Equal(t, "false", cve0002Row[4])
	assert.Equal(t, "", cve0002Row[5])
	assert.Equal(t, "RHSA-2024:0002", cve0002Row[9])
}

func TestExportCSV_Raw(t *testing.T) {
	findings := []*storage.ScanFinding{
		{
			Id:             "finding-1",
			CveName:        "CVE-2024-0001",
			AdvisoryId:     "RHSA-2024:0001",
			SourceName:     "Red Hat",
			Severity:       storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			Cvss:           9.8,
			NvdCvss:        9.8,
			EpssPercentile: 0.95,
			IsFixable:      true,
			FixedBy:        "1.2.3",
			DataSource:     "nvd",
			Description:    "Critical vulnerability",
		},
		{
			Id:             "finding-2",
			CveName:        "CVE-2024-0001",
			AdvisoryId:     "GHSA-xxxx-yyyy-zzzz",
			SourceName:     "GitHub",
			Severity:       storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
			Cvss:           8.5,
			NvdCvss:        9.8,
			EpssPercentile: 0.95,
			IsFixable:      true,
			FixedBy:        "1.2.3",
			DataSource:     "github",
			Description:    "Same CVE different advisory",
		},
	}

	var buf bytes.Buffer
	err := ExportCSV(&buf, findings, FormatRaw)
	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 2 rows (one per finding)
	require.Len(t, records, 3, "expected header + 2 finding rows")

	// Verify header
	expectedHeader := []string{
		"CVE", "Advisory ID", "Advisory Source", "Severity", "CVSS",
		"NVD CVSS", "EPSS", "Fixable", "Fixed In", "Data Source", "Description",
	}
	assert.Equal(t, expectedHeader, records[0])

	// Verify first finding
	assert.Equal(t, "CVE-2024-0001", records[1][0])
	assert.Equal(t, "RHSA-2024:0001", records[1][1])
	assert.Equal(t, "Red Hat", records[1][2])
	assert.Equal(t, "CRITICAL_VULNERABILITY_SEVERITY", records[1][3])
	assert.Equal(t, "9.80", records[1][4])
	assert.Equal(t, "9.80", records[1][5])
	assert.Equal(t, "0.95", records[1][6])
	assert.Equal(t, "true", records[1][7])
	assert.Equal(t, "1.2.3", records[1][8])
	assert.Equal(t, "nvd", records[1][9])
	assert.Equal(t, "Critical vulnerability", records[1][10])

	// Verify second finding (different advisory for same CVE)
	assert.Equal(t, "CVE-2024-0001", records[2][0])
	assert.Equal(t, "GHSA-xxxx-yyyy-zzzz", records[2][1])
	assert.Equal(t, "GitHub", records[2][2])
	assert.Equal(t, "IMPORTANT_VULNERABILITY_SEVERITY", records[2][3])
	assert.Equal(t, "8.50", records[2][4])
}

func TestExportCSV_EmptyFindings(t *testing.T) {
	var buf bytes.Buffer
	err := ExportCSV(&buf, []*storage.ScanFinding{}, FormatGrouped)
	require.NoError(t, err)

	// Should just have header
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 1, "expected only header row")
}

func TestExportCSV_DefaultFormat(t *testing.T) {
	findings := []*storage.ScanFinding{
		{
			Id:         "finding-1",
			CveName:    "CVE-2024-0001",
			AdvisoryId: "RHSA-2024:0001",
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			Cvss:       9.8,
		},
	}

	var buf bytes.Buffer
	// Use invalid format to test default behavior
	err := ExportCSV(&buf, findings, "invalid")
	require.NoError(t, err)

	// Should default to grouped format
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Verify it used grouped format (has "Advisories" column)
	assert.Contains(t, records[0], "Advisories")
}
