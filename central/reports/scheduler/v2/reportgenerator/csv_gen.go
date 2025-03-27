package reportgenerator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	csvHeader = []string{
		"Cluster",
		"Namespace",
		"Deployment",
		"Image",
		"Component",
		"CVE",
		"Fixable",
		"CVE Fixed In",
		"Severity",
		"CVSS",
		"Discovered At",
		"Reference",
	}
)

// GenerateCSV takes in the results of vuln report query, converts to CSV and returns zipped data
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string, reportFilters *storage.VulnerabilityReportFilters, optionalColumns *storage.VulnerabilityReportOptionalColumns) (*bytes.Buffer, error) {
	csvHeaderClone := addOptionalColumnstoHeader(optionalColumns)
	csvWriter := csv.NewGenericWriter(csvHeaderClone, true)
	for _, r := range cveResponses {
		row := csv.Value{
			r.GetCluster(),
			r.GetNamespace(),
			r.GetDeployment(),
			r.GetImage(),
			r.GetComponent(),
			r.GetCVE(),
			strconv.FormatBool(r.GetFixable()),
			r.GetFixedByVersion(),
			strings.ToTitle(stringutils.GetUpTo(r.GetSeverity().String(), "_")),
			strconv.FormatFloat(r.GetCVSS(), 'f', 2, 64),
			r.GetDiscoveredAtImage(),
			r.Link,
		}
		addOptionalColumnstoRow(optionalColumns, &row, csvWriter, r)
		csvWriter.AddValue(row)
	}

	var buf bytes.Buffer
	err := csvWriter.WriteBytes(&buf)
	if err != nil {
		return nil, errors.Wrap(err, "error creating csv report")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	truncatedName := configName
	if len(configName) > 80 {
		truncatedName = configName[0:80] + "..."
	}

	reportName := fmt.Sprintf("RHACS_Vulnerability_Report_%s_%s.csv", truncatedName, time.Now().Format("02_January_2006"))
	zipFile, err := zipWriter.Create(reportName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create the zip file for report config '%s'", configName)
	}
	_, err = zipFile.Write(buf.Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to write the zip file for report config '%s'", configName)
	}
	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to close the zip file for report config %s", configName)
	}
	return &zipBuf, nil
}

func addOptionalColumnstoHeader(optionalColumns *storage.VulnerabilityReportOptionalColumns) []string {
	csvHeaderClone := make([]string, len(csvHeader))
	copy(csvHeaderClone, csvHeader)
	if optionalColumns.GetIncludeNvdCvss() {
		csvHeaderClone = append(csvHeaderClone, "NVDCVSS")
	}
	if optionalColumns.GetIncludeEpssProbability() {
		csvHeaderClone = append(csvHeaderClone, "EPSS Probability Percentage")
	}
	return csvHeaderClone
}

func addOptionalColumnstoRow(optionalColumns *storage.VulnerabilityReportOptionalColumns, row *csv.Value, csvWriter *csv.GenericWriter, resp *ImageCVEQueryResponse) {
	if optionalColumns.GetIncludeNvdCvss() {
		csvWriter.AppendToValue(row, strconv.FormatFloat(resp.GetNVDCVSS(), 'f', 2, 64))
	}
	if optionalColumns.GetIncludeEpssProbability() {
		epssScore := resp.GetEPSSProbability()
		if epssScore != nil {
			csvWriter.AppendToValue(row, strconv.FormatFloat(*resp.GetEPSSProbability()*100, 'f', 3, 64))
		} else {
			csvWriter.AppendToValue(row, "Not Available")
		}
	}
}
