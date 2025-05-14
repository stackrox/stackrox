package format

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
)

const (
	emptyValue           = "Data not found for the cluster"
	successfulClusterFmt = "cluster_%s.csv"
	failedClusterFmt     = "failed_cluster_%s.csv"
)

var (
	csvHeader = []string{
		"Control Reference",
		"Check(CCR)",
		"Profile(version)",
		"Check Description",
		"Cluster",
		"Status",
		"Remediation",
		"Rationale",
		"Instructions",
	}
	failedClusterCSVHeader = []string{
		"Cluster ID",
		"Cluster Name",
		"Reason",
		"Compliance Operator Version",
	}
)

//go:generate mockgen-wrapper
type CSVWriter interface {
	AddValue(csv.Value)
	WriteCSV(io.Writer) error
}

//go:generate mockgen-wrapper
type ZipWriter interface {
	Create(string) (io.Writer, error)
	Close() error
}

type FormatterImpl struct {
	newZipWriter func(*bytes.Buffer) ZipWriter
	newCSVWriter func(csv.Header, bool) CSVWriter
}

func NewFormatter() *FormatterImpl {
	return &FormatterImpl{
		newZipWriter: createNewZipWriter,
		newCSVWriter: createNewCSVWriter,
	}
}

// FormatCSVReport generates zip data containing CSV files (one per cluster).
// If a cluster fails, the generated CSV file will contain the reason for the reason but (no check results).
// If a cluster success, the generated CSV file will contain all the check results with enhanced information (e.g. remediation, associated profile, etc)
// The results parameter is expected to contain the clusters that succeed (no failed clusters should be passed in results).
func (f *FormatterImpl) FormatCSVReport(results map[string][]*report.ResultRow, failedClusters map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) (buffRet *bytes.Buffer, errRet error) {
	var buf bytes.Buffer
	zipWriter := f.newZipWriter(&buf)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			buffRet = nil
			errRet = errors.Wrap(err, "unable to create a zip file of the compliance report")
		}
	}()
	for clusterID, failedCluster := range failedClusters {
		fileName := fmt.Sprintf(failedClusterFmt, clusterID)
		if err := f.createFailedClusterFileInZip(zipWriter, fileName, failedCluster); err != nil {
			return nil, errors.Wrap(err, "error creating failed cluster report")
		}
	}
	for clusterID, res := range results {
		// We should not receive results from a failed cluster
		if _, ok := failedClusters[clusterID]; ok {
			continue
		}
		fileName := fmt.Sprintf(successfulClusterFmt, clusterID)
		err := f.createCSVInZip(zipWriter, fileName, res)
		if err != nil {
			return nil, errors.Wrap(err, "error creating csv report")
		}
	}
	return &buf, nil
}

func (f *FormatterImpl) createCSVInZip(zipWriter ZipWriter, filename string, clusterResults []*report.ResultRow) error {
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	csvWriter := f.newCSVWriter(csvHeader, true)
	if len(clusterResults) != 0 {
		for _, checkRes := range clusterResults {
			csvWriter.AddValue(generateRecord(checkRes))
		}
	} else {
		csvWriter.AddValue([]string{emptyValue})
	}
	return csvWriter.WriteCSV(w)
}

func generateRecord(row *report.ResultRow) []string {
	// The order in the slice needs to match the order defined in `csvHeader`
	return []string{
		row.ControlRef,
		row.CheckName,
		row.Profile,
		row.Description,
		row.ClusterName,
		row.Status,
		row.Remediation,
		row.Rationale,
		row.Instructions,
	}
}

func (f *FormatterImpl) createFailedClusterFileInZip(zipWriter ZipWriter, filename string, failedCluster *storage.ComplianceOperatorReportSnapshotV2_FailedCluster) error {
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	csvWriter := f.newCSVWriter(failedClusterCSVHeader, true)
	csvWriter.AddValue(generateFailRecord(failedCluster))
	return csvWriter.WriteCSV(w)
}

func generateFailRecord(failedCluster *storage.ComplianceOperatorReportSnapshotV2_FailedCluster) []string {
	// The order in the slice needs to match the order defined in `failedClusterCSVHeader`
	return []string{
		failedCluster.GetClusterId(),
		failedCluster.GetClusterName(),
		failedCluster.GetReason(),
		failedCluster.GetOperatorVersion(),
	}
}

func createNewZipWriter(buf *bytes.Buffer) ZipWriter {
	return zip.NewWriter(buf)
}

func createNewCSVWriter(header csv.Header, sort bool) CSVWriter {
	return csv.NewGenericWriter(header, sort)
}
