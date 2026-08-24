package node

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSV_EmptyResponses(t *testing.T) {
	buf, err := generateCSV(nil, "test-report")
	require.NoError(t, err)
	require.NotNil(t, buf)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.Contains(t, reader.File[0].Name, "RHACS_Node_Vulnerability_Report_")
	assert.Contains(t, reader.File[0].Name, "test-report")
}

func TestGenerateCSV_WithResponses(t *testing.T) {
	cluster := "prod-us"
	node := "node-1"
	component := "openssl"
	componentVersion := "1.1.1"
	cve := "CVE-2021-1234"
	fixable := true
	fixedBy := "1.1.2"
	severity := storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	cvss := 9.8

	responses := []*NodeCVEQueryResponse{
		{
			Cluster:          &cluster,
			Node:             &node,
			Component:        &component,
			ComponentVersion: &componentVersion,
			CVE:              &cve,
			Fixable:          &fixable,
			FixedByVersion:   &fixedBy,
			Severity:         &severity,
			CVSS:             &cvss,
			Link:             "https://nvd.nist.gov/vuln/detail/CVE-2021-1234",
		},
	}

	buf, err := generateCSV(responses, "test-report")
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)

	rc, err := reader.File[0].Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	csvReader := csv.NewReader(rc)
	records, err := csvReader.ReadAll()
	require.NoError(t, err)

	// Header + 1 data row
	require.Len(t, records, 2)
	assert.Equal(t, csvHeader, records[0])
	assert.Equal(t, "prod-us", records[1][0])
	assert.Equal(t, "node-1", records[1][1])
	assert.Equal(t, "openssl", records[1][2])
	assert.Equal(t, "1.1.1", records[1][3])
	assert.Equal(t, "CVE-2021-1234", records[1][4])
	assert.Equal(t, "true", records[1][5])
	assert.Equal(t, "1.1.2", records[1][6])
	assert.Equal(t, "CRITICAL", records[1][7])
	assert.Equal(t, "9.80", records[1][8])
	assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1234", records[1][9])
}

func TestGenerateCSV_LongConfigNameIsTruncated(t *testing.T) {
	longName := ""
	for i := 0; i < 100; i++ {
		longName += "a"
	}
	buf, err := generateCSV(nil, longName)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.Contains(t, reader.File[0].Name, "...")
	assert.LessOrEqual(t, len(reader.File[0].Name), 200)
}

func TestGenerateCSV_ZipEntryHasModifiedTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Minute)
	buf, err := generateCSV(nil, "test")
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.False(t, reader.File[0].Modified.IsZero(), "ZIP entry Modified timestamp should not be zero")
	assert.True(t, reader.File[0].Modified.After(before), "ZIP entry Modified timestamp should be recent")
}

func TestGenerateCSV_NilFieldsProduceDefaults(t *testing.T) {
	responses := []*NodeCVEQueryResponse{
		{},
	}

	buf, err := generateCSV(responses, "defaults-test")
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	rc, err := reader.File[0].Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	csvReader := csv.NewReader(rc)
	records, err := csvReader.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)

	row := records[1]
	assert.Equal(t, "", row[0])        // Cluster
	assert.Equal(t, "", row[1])        // Node
	assert.Equal(t, "", row[2])        // Component
	assert.Equal(t, "", row[3])        // ComponentVersion
	assert.Equal(t, "", row[4])        // CVE
	assert.Equal(t, "false", row[5])   // Fixable
	assert.Equal(t, "", row[6])        // FixedBy
	assert.Equal(t, "UNKNOWN", row[7]) // Severity
	assert.Equal(t, "0.00", row[8])    // CVSS
	assert.Equal(t, "", row[9])        // Link
}
