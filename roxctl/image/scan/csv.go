package scan

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
)

type sortRecord struct {
	index    int
	severity storage.VulnerabilitySeverity
}

// PrintCSV prints image scan result in csv format
func PrintCSV(imageResult *storage.Image, out io.Writer) error {
	w := csv.NewWriter(out)
	w.UseCRLF = true

	defer w.Flush()

	header := []string{"CVE", "CVSS Score", "Severity Rating", "Summary", "Component", "Version", "Fixed By", "Layer Instruction"}
	if err := w.Write(header); err != nil {
		return errors.Wrap(err, "could not write CSV header")
	}

	layers := imageResult.GetMetadata().GetV1().GetLayers()
	components := imageResult.GetScan().GetComponents()

	// Sort components by layerIndex
	sort.SliceStable(components, func(p, q int) bool { return components[p].GetLayerIndex() < components[q].GetLayerIndex() })

	var currentLayerIndex int32 = -1
	var records [][]string
	var sortRecords []sortRecord

	for _, component := range components {
		if len(component.GetVulns()) == 0 {
			continue
		}

		if currentLayerIndex != component.GetLayerIndex() {
			if err := sortAndPrint(w, records, sortRecords); err != nil {
				return err
			}
			// Clear the current record
			sortRecords = sortRecords[:0]
			records = records[:0]
			currentLayerIndex = component.GetLayerIndex()
		}
		for _, vuln := range component.GetVulns() {
			sortRecords = append(sortRecords, sortRecord{len(records), vuln.GetSeverity()})
			layer := layers[component.GetLayerIndex()]
			formattedSeverity := cvss.FormatSeverity(vuln.GetSeverity())
			records = append(records, []string{vuln.GetCve(), fmt.Sprintf("%g", vuln.GetCvss()), formattedSeverity, vuln.GetSummary(), component.GetName(), component.GetVersion(), vuln.GetFixedBy(), layer.GetInstruction() + " " + layer.GetValue()})
		}
	}
	return sortAndPrint(w, records, sortRecords)
}

func sortAndPrint(w *csv.Writer, records [][]string, sortRecords []sortRecord) error {
	sort.SliceStable(sortRecords, func(p, q int) bool { return sortRecords[p].severity > sortRecords[q].severity })
	for _, sr := range sortRecords {
		if err := w.Write(records[sr.index]); err != nil {
			return errors.Wrap(err, "could not write CSV record")
		}
	}
	return nil
}
