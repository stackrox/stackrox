package reportgenerator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/features"
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
		"CVE Origin",
	}

	// originDisplayNames maps VulnOrigin enum values to human-readable names for the
	// CSV report. This must be kept in sync with the equivalent UI map.
	originDisplayNames = map[storage.VulnOrigin]string{
		storage.VulnOrigin_VULN_ORIGIN_ALPINE:  "Alpine Linux",
		storage.VulnOrigin_VULN_ORIGIN_AMAZON:  "Amazon Linux",
		storage.VulnOrigin_VULN_ORIGIN_DEBIAN:  "Debian",
		storage.VulnOrigin_VULN_ORIGIN_ORACLE:  "Oracle Linux",
		storage.VulnOrigin_VULN_ORIGIN_OSV:     "OSV.dev",
		storage.VulnOrigin_VULN_ORIGIN_PHOTON:  "Photon OS",
		storage.VulnOrigin_VULN_ORIGIN_RED_HAT: "Red Hat",
		storage.VulnOrigin_VULN_ORIGIN_SUSE:    "SUSE",
		storage.VulnOrigin_VULN_ORIGIN_UBUNTU:  "Ubuntu",
		storage.VulnOrigin_VULN_ORIGIN_OTHER:   "Other",
	}
)

func originDisplayName(origin storage.VulnOrigin) string {
	if name, ok := originDisplayNames[origin]; ok {
		return name
	}
	return origin.String()
}

func formatCSVRow(r *ImageCVEQueryResponse) []string {
	var epssScore string
	if r.GetEPSSProbability() != nil {
		epssScore = strconv.FormatFloat(*r.GetEPSSProbability()*100, 'f', 3, 64)
	} else {
		epssScore = "Not Available"
	}

	csvRow := []string{
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
		strconv.FormatFloat(r.GetCVSS(), 'f', 2, 64),
		strconv.FormatFloat(r.GetNVDCVSS(), 'f', 2, 64),
		epssScore}

	if features.KnownExploitedVulnerabilities.Enabled() {
		var cisaKev string
		if r.GetCisaKev() != nil {
			cisaKev = strconv.FormatBool(*r.GetCisaKev())
		} else {
			cisaKev = "Not Available"
		}
		csvRow = append(csvRow, cisaKev)
	}

	csvRow = append(csvRow,
		r.GetDiscoveredAtImage(),
		r.Link,
		r.GetAdvisoryName(),
		r.GetAdvisoryLink(),
		originDisplayName(r.GetOrigin()),
	)

	return csvRow
}

func formatCol() []string {
	csvHeaderCols := csvHeader
	if features.KnownExploitedVulnerabilities.Enabled() {
		csvHeaderCols = append(csvHeaderCols[:0:0], csvHeader...)
		epssIdx := slices.Index(csvHeaderCols, "EPSS Probability Percentage")
		csvHeaderCols = slices.Insert(csvHeaderCols, epssIdx+1, "CISA KEV")
	}
	return csvHeaderCols
}

func csvReportName(configName string) string {
	truncatedName := configName
	if len(configName) > 80 {
		truncatedName = configName[0:80] + "..."
	}
	now := time.Now()
	return fmt.Sprintf("RHACS_Vulnerability_Report_%s_%s.csv", truncatedName, now.Format("02_January_2006"))
}

// GenerateCSV takes in the results of vuln report query, converts to CSV and returns zipped data
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string) (*bytes.Buffer, error) {
	csvWriter := csv.NewGenericWriter(formatCol(), true)

	for _, r := range cveResponses {
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
			strconv.FormatFloat(r.GetCVSS(), 'f', 2, 64),
			strconv.FormatFloat(r.GetNVDCVSS(), 'f', 2, 64),
			epssScore,
		}
		if features.KnownExploitedVulnerabilities.Enabled() {
			var cisaKev string
			if r.GetCisaKev() != nil {
				cisaKev = strconv.FormatBool(*r.GetCisaKev())
			} else {
				cisaKev = "Not Available"
			}
			row = append(row, cisaKev)
		}
		row = append(row,
			r.GetDiscoveredAtImage(),
			r.Link,
			r.GetAdvisoryName(),
			r.GetAdvisoryLink(),
			originDisplayName(r.GetOrigin()),
		)
		csvWriter.AddValue(row)
	}

	var buf bytes.Buffer
	err := csvWriter.WriteBytes(&buf)
	if err != nil {
		return nil, errors.Wrap(err, "error creating csv report")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	header := &zip.FileHeader{
		Name:     csvReportName(configName),
		Method:   zip.Deflate,
		Modified: time.Now(),
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
