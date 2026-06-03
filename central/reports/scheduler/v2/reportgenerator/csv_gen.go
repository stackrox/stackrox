package reportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	gocsv "encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/stringutils"
)

const chunkSize = 5000

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

	csvColCount = len(csvHeader)
)

// GenerateCSV takes in the results of vuln report query, converts to CSV and returns zipped data
func GenerateCSV(cveResponses []*ImageCVEQueryResponse, configName string) (*bytes.Buffer, error) {
	// add header for component version
	csvWriter := csv.NewGenericWriter(csvHeader, true)

	for _, r := range cveResponses {
		csvWriter.AddValue(formatRow(r))
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

func formatRow(r *ImageCVEQueryResponse) csv.Value {
	var epssScore string
	if r.GetEPSSProbability() != nil {
		epssScore = strconv.FormatFloat(*r.GetEPSSProbability()*100, 'f', 3, 64)
	} else {
		epssScore = "Not Available"
	}
	return csv.Value{
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
		r.GetDiscoveredAtImage(),
		r.Link,
		r.GetAdvisoryName(),
		r.GetAdvisoryLink(),
	}
}

// linkColumnIdx is the index of the "Reference" (CVE link) column in csvHeader.
const linkColumnIdx = 14

// formatRowInto writes the CSV fields for r into the pre-allocated row slice.
// The caller must ensure len(row) >= csvColCount.
func formatRowInto(r *ImageCVEQueryResponse, row csv.Value) {
	var epssScore string
	if r.GetEPSSProbability() != nil {
		epssScore = strconv.FormatFloat(*r.GetEPSSProbability()*100, 'f', 3, 64)
	} else {
		epssScore = "Not Available"
	}
	row[0] = r.GetCluster()
	row[1] = r.GetNamespace()
	row[2] = r.GetDeployment()
	row[3] = r.GetImage()
	row[4] = r.GetComponent()
	row[5] = r.GetComponentVersion()
	row[6] = r.GetCVE()
	row[7] = strconv.FormatBool(r.GetFixable())
	row[8] = r.GetFixedByVersion()
	row[9] = strings.ToTitle(stringutils.GetUpTo(r.GetSeverity().String(), "_"))
	row[10] = strconv.FormatFloat(r.GetCVSS(), 'f', 2, 64)
	row[11] = strconv.FormatFloat(r.GetNVDCVSS(), 'f', 2, 64)
	row[12] = epssScore
	row[13] = r.GetDiscoveredAtImage()
	row[14] = r.Link
	row[15] = r.GetAdvisoryName()
	row[16] = r.GetAdvisoryLink()
}

// csvBatchWriter writes batches of rows to CSV with pre-allocated row buffers.
// CVE reference links are resolved per batch via the provided resolver and
// accumulated across batches so each CVE ID is fetched at most once.
type csvBatchWriter struct {
	csvW         *gocsv.Writer
	resolveLinks func(ctx context.Context, newIDs []string) (map[string]string, error)
	linkMap      map[string]string
	rowBuf       csv.Value
	newIDs       []string
}

func newCSVBatchWriter(csvW *gocsv.Writer, resolveLinks func(ctx context.Context, newIDs []string) (map[string]string, error)) *csvBatchWriter {
	return &csvBatchWriter{
		csvW:         csvW,
		resolveLinks: resolveLinks,
		linkMap:      make(map[string]string),
		rowBuf:       make(csv.Value, csvColCount),
		newIDs:       make([]string, 0, chunkSize),
	}
}

func (bw *csvBatchWriter) writeBatch(ctx context.Context, batch []*ImageCVEQueryResponse) error {
	bw.newIDs = bw.newIDs[:0]
	for _, r := range batch {
		if id := r.GetCVEID(); id != "" {
			if _, ok := bw.linkMap[id]; !ok {
				bw.newIDs = append(bw.newIDs, id)
			}
		}
	}

	if len(bw.newIDs) > 0 {
		resolved, err := bw.resolveLinks(ctx, bw.newIDs)
		if err != nil {
			return err
		}
		for k, v := range resolved {
			bw.linkMap[k] = v
		}
	}

	for _, r := range batch {
		formatRowInto(r, bw.rowBuf)
		if link, ok := bw.linkMap[r.GetCVEID()]; ok {
			bw.rowBuf[linkColumnIdx] = link
		}
		if err := bw.csvW.Write(bw.rowBuf); err != nil {
			return err
		}
	}
	return nil
}

// GenerateCSVStreaming streams rows from cursor-based batch query callbacks into
// a zipped CSV buffer. CVE reference links are resolved per batch using the
// provided resolver, which receives a context with the cursor's transaction so
// it can share the same DB connection.
func GenerateCSVStreaming(
	configName string,
	resolveLinks func(ctx context.Context, cveIDs []string) (map[string]string, error),
	runDeployedQuery func(batchFn func(ctx context.Context, batch []*ImageCVEQueryResponse) error) error,
	runWatchedQuery func(batchFn func(ctx context.Context, batch []*ImageCVEQueryResponse) error) error,
) (*StreamingReportResult, error) {
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	truncatedName := configName
	if len(configName) > 80 {
		truncatedName = configName[0:80] + "..."
	}
	now := time.Now()
	reportName := fmt.Sprintf("RHACS_Vulnerability_Report_%s_%s.csv",
		truncatedName, now.Format("02_January_2006"))
	zipFile, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:     reportName,
		Method:   zip.Deflate,
		Modified: now,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create zip file for report config '%s'", configName)
	}

	csvW := gocsv.NewWriter(zipFile)
	csvW.UseCRLF = true
	if err := csvW.Write(csvHeader); err != nil {
		return nil, errors.Wrap(err, "error writing CSV header")
	}

	bw := newCSVBatchWriter(csvW, resolveLinks)
	result := &StreamingReportResult{}

	if runDeployedQuery != nil {
		err = runDeployedQuery(func(ctx context.Context, batch []*ImageCVEQueryResponse) error {
			result.NumDeployedImageResults += len(batch)
			return bw.writeBatch(ctx, batch)
		})
		if err != nil {
			return nil, errors.Wrap(err, "error streaming deployed image rows")
		}
	}

	if runWatchedQuery != nil {
		err = runWatchedQuery(func(ctx context.Context, batch []*ImageCVEQueryResponse) error {
			result.NumWatchedImageResults += len(batch)
			return bw.writeBatch(ctx, batch)
		})
		if err != nil {
			return nil, errors.Wrap(err, "error streaming watched image rows")
		}
	}

	csvW.Flush()
	if err := csvW.Error(); err != nil {
		return nil, errors.Wrap(err, "error flushing CSV writer")
	}
	if err := zipWriter.Close(); err != nil {
		return nil, errors.Wrapf(err, "unable to close zip file for report config '%s'", configName)
	}

	result.ZippedCSVData = &zipBuf
	return result, nil
}
