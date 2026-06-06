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
		"Component Version",
		"CVE",
		"Fixable",
		"CVE Fixed In",
		"Severity",
		"CVSS",
		"NVDCVSS",
		"EPSS Probability Percentage",
		"Discovered At",
		"Reference",
		"Advisory Name",
		"Advisory Link",
	}
)

// GenerateCSV takes in the results of vuln report query, converts to CSV and returns zipped data
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string) (*bytes.Buffer, error) {
	// add header for component version
	csvWriter := csv.NewGenericWriter(csvHeader, true)

	for _, r := range cveResponses {
		var cvssScore string
		if r.GetCVSS() != nil {
			cvssScore = strconv.FormatFloat(*r.GetCVSS(), 'f', 2, 64)
		} else {
			cvssScore = "Not Available"
		}
		var nvdCVSSScore string
		if r.GetNVDCVSS() != nil {
			nvdCVSSScore = strconv.FormatFloat(*r.GetNVDCVSS(), 'f', 2, 64)
		} else {
			nvdCVSSScore = "Not Available"
		}
		var epssScore string
		if r.GetEPSSProbability() != nil {
			epssScore = strconv.FormatFloat(*r.GetEPSSProbability()*100, 'f', 3, 64)
		} else {
			epssScore = "Not Available"
		}
		row := csv.Value{
			r.GetCluster(),
			r.GetNamespace(),
			r.GetDeployment(),
			r.GetImage(),
			r.GetComponent(),
			r.GetComponentVersion(),
			r.GetCVE(),
			strconv.FormatBool(r.GetFixable()),
			r.GetFixedByVersion(),
			strings.ToTitle(stringutils.GetUpTo(r.GetSeverity().String(), "_")),
			cvssScore,
			nvdCVSSScore,
			epssScore,
			r.GetDiscoveredAtImage(),
			r.Link,
			r.GetAdvisoryName(),
			r.GetAdvisoryLink(),
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

	now := time.Now()
	reportName := fmt.Sprintf("RHACS_Vulnerability_Report_%s_%s.csv", truncatedName, now.Format("02_January_2006"))
	header := &zip.FileHeader{
		Name:     reportName,
		Method:   zip.Deflate,
		Modified: now,
	}
	zipFile, err := zipWriter.CreateHeader(header)
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
