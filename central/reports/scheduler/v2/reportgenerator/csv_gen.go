package reportgenerator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
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
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string, includeNVDCVSS bool) (*bytes.Buffer, error) {
	csvHeaderClone := csvHeader
	if includeNVDCVSS {
		csvHeaderClone = append(csvHeader, "NVDCVSS")
	}
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
		if includeNVDCVSS {
			csvWriter.AppendToValue(&row, strconv.FormatFloat(r.GetNVDCVSS(), 'f', 2, 64))
		}
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
