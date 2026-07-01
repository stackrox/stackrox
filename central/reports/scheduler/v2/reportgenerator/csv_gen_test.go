package reportgenerator

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
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
	cvss := 9.8
	nvdcvss := 9.5
	epss := 0.95
	discovered := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	advisoryName := "RHSA-2024:1234"
	advisoryLink := "https://access.redhat.com/errata/RHSA-2024:1234"

	r := &ImageCVEQueryResponse{
		Cluster:           &cluster,
		Namespace:         &ns,
		Deployment:        &deploy,
		Image:             &image,
		Component:         &component,
		ComponentVersion:  &compVersion,
		CVE:               &cve,
		Fixable:           &fixable,
		FixedByVersion:    &fixedBy,
		Severity:          &severity,
		CVSS:              &cvss,
		NVDCVSS:           &nvdcvss,
		EPSSProbability:   &epss,
		DiscoveredAtImage: &discovered,
		AdvisoryName:      &advisoryName,
		AdvisoryLink:      &advisoryLink,
		Link:              "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
	}

	row := formatCSVRow(r)
	assert.Equal(t, len(csvHeader), len(row), "row should have same number of columns as header")
	assert.Equal(t, "test-cluster", row[0])
	assert.Equal(t, "test-ns", row[1])
	assert.Equal(t, "test-deploy", row[2])
	assert.Equal(t, "CVE-2024-1234", row[6])
	assert.Equal(t, "true", row[7])
	assert.Equal(t, "CRITICAL", row[9])
	assert.Equal(t, "95.000", row[12])
	assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", row[14])
}

func TestFormatCSVRow_NilFields(t *testing.T) {
	r := &ImageCVEQueryResponse{}
	row := formatCSVRow(r)
	assert.Equal(t, len(csvHeader), len(row))
	assert.Equal(t, "", row[0])
	assert.Equal(t, "Not Available", row[12])
	assert.Equal(t, "Not Available", row[13])
}

func TestCsvReportName(t *testing.T) {
	name := csvReportName("My Report")
	assert.Contains(t, name, "RHACS_Vulnerability_Report_My Report_")
	assert.Contains(t, name, ".csv")

	longName := ""
	for i := 0; i < 100; i++ {
		longName += "a"
	}
	name = csvReportName(longName)
	assert.Contains(t, name, "...")
}
