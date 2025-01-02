package format

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/pkg/csv"
)

const (
	EmptyValue = "Data not found for the cluster"
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

func (f *FormatterImpl) FormatCSVReport(results map[string][]*report.ResultRow) (buffRet *bytes.Buffer, errRet error) {
	var buf bytes.Buffer
	zipWriter := f.newZipWriter(&buf)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			buffRet = nil
			errRet = errors.Wrap(err, "unable to create a zip file of the compliance report")
		}
	}()
	for clusterID, res := range results {
		fileName := fmt.Sprintf("cluster_%s.csv", clusterID)
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
		csvWriter.AddValue([]string{EmptyValue})
	}
	return csvWriter.WriteCSV(w)
}

func generateRecord(row *report.ResultRow) []string {
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

func createNewZipWriter(buf *bytes.Buffer) ZipWriter {
	return zip.NewWriter(buf)
}

func createNewCSVWriter(header csv.Header, sort bool) CSVWriter {
	return csv.NewGenericWriter(header, sort)
}
