package reportgenerator

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSV_ZipEntryHasModifiedTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Minute)
	buf, err := GenerateCSV(nil, "test")
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.False(t, reader.File[0].Modified.IsZero(), "ZIP entry Modified timestamp should not be zero")
	assert.True(t, reader.File[0].Modified.After(before), "ZIP entry Modified timestamp should be recent")
}

func TestGenerateCSV_VulnerabilityStates(t *testing.T) {
	observed := storage.VulnerabilityState_OBSERVED
	deferred := storage.VulnerabilityState_DEFERRED
	falsePositive := storage.VulnerabilityState_FALSE_POSITIVE
	responses := []*ImageCVEQueryResponse{
		{State: &observed},
		{State: &deferred},
		{State: &falsePositive},
		{},
	}
	expectedStates := []string{"Observed", "Deferred", "False Positive", "Not Available"}

	for name, enabled := range map[string]bool{"KEV disabled": false, "KEV enabled": true} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), strconv.FormatBool(enabled))

			buf, err := GenerateCSV(responses, "test")
			require.NoError(t, err)

			reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			require.NoError(t, err)
			require.Len(t, reader.File, 1)
			file, err := reader.File[0].Open()
			require.NoError(t, err)
			defer func() { assert.NoError(t, file.Close()) }()

			rows, err := csv.NewReader(file).ReadAll()
			require.NoError(t, err)
			require.Len(t, rows, len(expectedStates)+1)
			assert.Equal(t, formatCol(), rows[0])
			stateColumn := slices.Index(rows[0], "Vulnerability State")
			require.NotEqual(t, -1, stateColumn)
			for i, state := range expectedStates {
				require.Len(t, rows[i+1], len(rows[0]))
				assert.Equal(t, state, rows[i+1][stateColumn])
			}
		})
	}
}

func TestGenerateCSV_FixDates(t *testing.T) {
	firstFixDate := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	secondFixDate := time.Date(2024, 8, 2, 0, 0, 0, 0, time.UTC)
	responses := []*ImageCVEQueryResponse{
		{CVEFixAvailableTimestamp: &firstFixDate},
		{CVEFixAvailableTimestamp: &secondFixDate},
		{},
	}
	expectedDates := []string{"July 01, 2024", "August 02, 2024", "Not Available"}

	for name, enabled := range map[string]bool{"KEV disabled": false, "KEV enabled": true} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), strconv.FormatBool(enabled))

			buf, err := GenerateCSV(responses, "test")
			require.NoError(t, err)
			reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			require.NoError(t, err)
			require.Len(t, reader.File, 1)
			file, err := reader.File[0].Open()
			require.NoError(t, err)
			defer func() { assert.NoError(t, file.Close()) }()

			rows, err := csv.NewReader(file).ReadAll()
			require.NoError(t, err)
			require.Len(t, rows, len(expectedDates)+1)
			assert.Equal(t, formatCol(), rows[0])
			dateColumn := slices.Index(rows[0], "Fix Date")
			require.NotEqual(t, -1, dateColumn)
			assert.Equal(t, slices.Index(rows[0], "CVE Fixed In")+1, dateColumn)
			for i, expected := range expectedDates {
				require.Len(t, rows[i+1], len(rows[0]))
				assert.Equal(t, expected, rows[i+1][dateColumn])
			}
		})
	}
}

func TestFormatCSVRow(t *testing.T) {
	cluster := "test-cluster"
	ns := "test-ns"
	deploy := "test-deploy"
	image := "quay.io/test:latest"
	component := "openssl"
	compVersion := "1.1.1"
	cve := "CVE-2024-1234"
	fixable := true
	fixedBy := "1.1.2"
	severity := storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	state := storage.VulnerabilityState_OBSERVED
	cvss := 9.8
	nvdcvss := 9.5
	epss := 0.95
	discovered := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	fixDate := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	advisoryName := "RHSA-2024:1234"
	advisoryLink := "https://access.redhat.com/errata/RHSA-2024:1234"

	r := &ImageCVEQueryResponse{
		Cluster:                  &cluster,
		Namespace:                &ns,
		Deployment:               &deploy,
		Image:                    &image,
		Component:                &component,
		ComponentVersion:         &compVersion,
		CVE:                      &cve,
		Fixable:                  &fixable,
		FixedByVersion:           &fixedBy,
		Severity:                 &severity,
		State:                    &state,
		CVSS:                     &cvss,
		NVDCVSS:                  &nvdcvss,
		EPSSProbability:          &epss,
		DiscoveredAtImage:        &discovered,
		CVEFixAvailableTimestamp: &fixDate,
		AdvisoryName:             &advisoryName,
		AdvisoryLink:             &advisoryLink,
		Link:                     "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
	}

	t.Run("KEV disabled", func(t *testing.T) {
		t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), "false")

		row := formatCSVRow(r)
		assert.Equal(t, len(formatCol()), len(row), "row should have same number of columns as header")
		assert.Equal(t, "test-cluster", row[0])
		assert.Equal(t, "test-ns", row[1])
		assert.Equal(t, "test-deploy", row[2])
		assert.Equal(t, "CVE-2024-1234", row[6])
		assert.Equal(t, "true", row[7])
		assert.Equal(t, "CRITICAL", row[10])
		assert.Equal(t, "95.000", row[13])
		assert.Equal(t, "July 01, 2024", row[slices.Index(formatCol(), "Fix Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Image Created Date")])
		assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", row[slices.Index(formatCol(), "Reference")])
		assert.Equal(t, "Observed", row[slices.Index(formatCol(), "Vulnerability State")])
	})

	t.Run("KEV enabled", func(t *testing.T) {
		t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), "true")

		// The two KEV columns are inserted after EPSS (index 13), shifting all
		// trailing columns right by two.
		row := formatCSVRow(r)
		assert.Equal(t, len(formatCol()), len(row), "row should have same number of columns as header")
		assert.Equal(t, "test-cluster", row[0])
		assert.Equal(t, "95.000", row[13])
		assert.Equal(t, "Not Available", row[14]) // CISA KEV (nil → "Not Available")
		assert.Equal(t, "Not Available", row[15]) // Known Ransomware Campaign (nil → "Not Available")
		assert.Equal(t, "July 01, 2024", row[slices.Index(formatCol(), "Fix Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Image Created Date")])
		assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", row[slices.Index(formatCol(), "Reference")])
		assert.Equal(t, "Observed", row[slices.Index(formatCol(), "Vulnerability State")])
	})
}

func TestFormatCSVRow_NilFields(t *testing.T) {
	r := &ImageCVEQueryResponse{}

	t.Run("KEV disabled", func(t *testing.T) {
		t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), "false")

		row := formatCSVRow(r)
		assert.Equal(t, len(formatCol()), len(row))
		assert.Equal(t, "", row[0])
		assert.Equal(t, "Not Available", row[13]) // EPSS Probability Percentage
		assert.Equal(t, "Not Available", row[14]) // Discovered At
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Fix Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Image Created Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Vulnerability State")])
	})

	t.Run("KEV enabled", func(t *testing.T) {
		t.Setenv(features.KnownExploitedVulnerabilities.EnvVar(), "true")

		row := formatCSVRow(r)
		assert.Equal(t, len(formatCol()), len(row))
		assert.Equal(t, "", row[0])
		assert.Equal(t, "Not Available", row[13]) // EPSS Probability Percentage
		assert.Equal(t, "Not Available", row[14]) // CISA KEV
		assert.Equal(t, "Not Available", row[15]) // Known Ransomware Campaign
		assert.Equal(t, "Not Available", row[16]) // Discovered At
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Fix Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Image Created Date")])
		assert.Equal(t, "Not Available", row[slices.Index(formatCol(), "Vulnerability State")])
	})
}

func TestCsvReportName(t *testing.T) {
	name := csvReportName("My Report")
	assert.Contains(t, name, "RHACS_Vulnerability_Report_My Report_")
	assert.Contains(t, name, ".csv")

	longName := strings.Repeat("a", 100)
	name = csvReportName(longName)
	assert.Contains(t, name, "...")
}
