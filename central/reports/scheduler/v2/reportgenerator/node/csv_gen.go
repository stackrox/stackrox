package node

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

var csvHeader = []string{
	"Cluster",
	"Node",
	"Component",
	"Component Version",
	"CVE",
	"Fixable",
	"CVE Fixed In",
	"Severity",
	"CVSS",
	"Reference",
}

func generateCSV(cveResponses []*NodeCVEQueryResponse, configName string) (*bytes.Buffer, error) {
	csvWriter := csv.NewGenericWriter(csvHeader, true)

	for _, r := range cveResponses {
		row := csv.Value{
			r.GetCluster(),
			r.GetNode(),
			r.GetComponent(),
			r.GetComponentVersion(),
			r.GetCVE(),
			strconv.FormatBool(r.GetFixable()),
			r.GetFixedByVersion(),
			strings.ToTitle(stringutils.GetUpTo(r.GetSeverity().String(), "_")),
			strconv.FormatFloat(r.GetCVSS(), 'f', 2, 64),
			r.Link,
		}
		csvWriter.AddValue(row)
	}

	var buf bytes.Buffer
	if err := csvWriter.WriteBytes(&buf); err != nil {
		return nil, errors.Wrap(err, "error creating csv report")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	truncatedName := configName
	if len(configName) > 80 {
		truncatedName = configName[0:80] + "..."
	}

	now := time.Now()
	reportName := fmt.Sprintf("RHACS_Node_Vulnerability_Report_%s_%s.csv", truncatedName, now.Format("02_January_2006"))
	header := &zip.FileHeader{
		Name:     reportName,
		Method:   zip.Deflate,
		Modified: now,
	}
	zipFile, err := zipWriter.CreateHeader(header)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create the zip file for report config '%s'", configName)
	}
	if _, err = zipFile.Write(buf.Bytes()); err != nil {
		return nil, errors.Wrapf(err, "unable to write the zip file for report config '%s'", configName)
	}
	if err = zipWriter.Close(); err != nil {
		return nil, errors.Wrapf(err, "unable to close the zip file for report config %s", configName)
	}
	return &zipBuf, nil
}
