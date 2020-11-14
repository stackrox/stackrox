package scan

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"

	"github.com/stackrox/rox/generated/storage"
)

type sortRecord struct {
	index     int
	cvssScore float32
}

// PrintCSV prints image scan result in csv format
func PrintCSV(imageResult *storage.Image) error {
	w := csv.NewWriter(os.Stdout)
	w.UseCRLF = true

	defer w.Flush()

	header := []string{"CVE", "CVSS Score", "Summary", "Component", "Version", "Fixed By", "Layer Instruction"}
	if err := w.Write(header); err != nil {
		return err
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
			sortRecords = append(sortRecords, sortRecord{len(records), vuln.GetCvss()})
			layer := layers[component.GetLayerIndex()]
			records = append(records, []string{vuln.GetCve(), fmt.Sprintf("%g", vuln.GetCvss()), vuln.GetSummary(), component.GetName(), component.GetVersion(), vuln.GetFixedBy(), layer.GetInstruction() + " " + layer.GetValue()})
		}
	}
	return sortAndPrint(w, records, sortRecords)
}

func sortAndPrint(w *csv.Writer, records [][]string, sortRecords []sortRecord) error {
	sort.SliceStable(sortRecords, func(p, q int) bool { return sortRecords[p].cvssScore > sortRecords[q].cvssScore })
	for _, sr := range sortRecords {
		if err := w.Write(records[sr.index]); err != nil {
			return err
		}
	}
	return nil
}
