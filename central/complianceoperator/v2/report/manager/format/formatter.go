package format

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/pkg/csv"
	"google.golang.org/protobuf/types/known/timestamppb"
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
func (f *FormatterImpl) FormatCSVReport(results map[string][]*report.ResultRow, clusters map[string]*report.ClusterData) (buffRet *bytes.Buffer, errRet error) {
	var buf bytes.Buffer
	zipWriter := f.newZipWriter(&buf)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			buffRet = nil
			errRet = errors.Wrap(err, "unable to create a zip file of the compliance report")
		}
	}()
	timestamp := timestamppb.Now()
	for clusterID, cluster := range clusters {
		if cluster.FailedInfo != nil {
			fileName := getFileName(failedClusterFmt, cluster.ClusterName, timestamp)
			if err := f.createFailedClusterFileInZip(zipWriter, fileName, cluster.FailedInfo); err != nil {
				return nil, errors.Wrap(err, "error creating failed cluster report")
			}
		}
		if len(results[clusterID]) == 0 && cluster.FailedInfo != nil {
			continue
		}
		if _, ok := results[clusterID]; !ok {
			return nil, errors.Errorf("found no results for cluster %q", clusterID)
		}
		fileName := getFileName(successfulClusterFmt, cluster.ClusterName, timestamp)
		if err := f.createCSVInZip(zipWriter, fileName, results[clusterID]); err != nil {
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

func (f *FormatterImpl) createFailedClusterFileInZip(zipWriter ZipWriter, filename string, failedCluster *report.FailedCluster) error {
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	csvWriter := f.newCSVWriter(failedClusterCSVHeader, true)
	for _, reason := range failedCluster.Reasons {
		// The order in the slice needs to match the order defined in `failedClusterCSVHeader`
		csvWriter.AddValue([]string{failedCluster.ClusterId, failedCluster.ClusterName, reason, failedCluster.OperatorVersion})
	}
	return csvWriter.WriteCSV(w)
}

func getFileName(format string, clusterName string, timestamp *timestamppb.Timestamp) string {
	year, month, day := timestamp.AsTime().Date()
	return fmt.Sprintf(format, fmt.Sprintf("%s_%d-%d-%d", clusterName, year, month, day))
}

func createNewZipWriter(buf *bytes.Buffer) ZipWriter {
	return zip.NewWriter(buf)
}

func createNewCSVWriter(header csv.Header, sort bool) CSVWriter {
	return csv.NewGenericWriter(header, sort)
}
