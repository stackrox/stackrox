package reportgenerator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/branding"
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
		"Version",
		"CVE",
		"Fixable",
		"CVE Fixed In",
		"Severity",
		"CVSS",
		"Discovered At",
		"Reference",
	}
)

// GenerateCSV takes in the results of vuln report query, converts to CSV and returns zipped CSV data
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string) (*bytes.Buffer, error) {
	csvWriter := csv.NewGenericWriter(csvHeader, true)
	for _, r := range cveResponses {
		discoveredTs := "Not Available"
		if r.DiscoveredAtImage != nil {
			discoveredTs = r.DiscoveredAtImage.Format("January 02, 2006")
		}
		csvWriter.AddValue(csv.Value{
			r.Cluster,
			r.Namespace,
			r.Deployment,
			r.Image,
			r.Component,
			r.ComponentVersion,
			r.CVE,
			strconv.FormatBool(r.Fixable),
			r.FixedByVersion,
			strings.ToTitle(stringutils.GetUpTo(r.Severity.String(), "_")),
			strconv.FormatFloat(r.CVSS, 'f', 2, 64),
			discoveredTs,
			r.Link,
		})
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

	reportName := fmt.Sprintf("%s_Vulnerability_Report_%s_%s.csv", branding.GetProductNameShort(),
		truncatedName, time.Now().Format("02_January_2006"))
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
