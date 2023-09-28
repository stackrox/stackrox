package openvex

import (
	"encoding/json"

	"github.com/hashicorp/go-multierror"
	"github.com/openvex/go-vex/pkg/vex"
	"github.com/stackrox/rox/generated/storage"
)

// Filter will filter vulnerabilities on the image based on input of the associated VEX report.
// If no VEX report is given, it's a no-op.
// When any vulnerability will be filtered, returns true.
func Filter(img *storage.Image) (bool, error) {
	// Nothing to filter.
	if len(img.GetOpenVexReport()) == 0 {
		return false, nil
	}
	var vexReports []*vex.VEX
	var unmarshalErrors *multierror.Error

	for _, storageReport := range img.GetOpenVexReport() {
		var report *vex.VEX
		if err := json.Unmarshal(storageReport.GetOpenVexReport(), report); err != nil {
			unmarshalErrors = multierror.Append(unmarshalErrors, err)
			continue
		}
		vexReports = append(vexReports, report)
	}
	var filtered bool
	for _, component := range img.GetScan().GetComponents() {
		for _, report := range vexReports {
			if filterComponentVulnerabilities(component, report) {
				filtered = true
			}
		}
	}
	return filtered, nil
}

func filterComponentVulnerabilities(component *storage.EmbeddedImageScanComponent, report *vex.VEX) bool {
	// Go through all vulnerabilities associated with the component.
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(component.GetVulns()))
	var filtered bool
	for _, vuln := range component.GetVulns() {
		statements := report.StatementsByVulnerability(vuln.GetCve())
		// If no statement is within the report for the particular CVE, skip it.
		if len(statements) == 0 {
			vulns = append(vulns, vuln)
			continue
		}
		// There can only be one statement per CVE. If its either marked as not affected or fixed, we will filter
		// it out.
		switch statements[0].Status {
		case vex.StatusNotAffected, vex.StatusFixed:
			log.Infof("Filtering out CVE %s since it is marked as %q in VEX report",
				vuln.GetCve(), statements[0].Status)
			filtered = true
		default:
			vulns = append(vulns, vuln)
		}
	}
	component.Vulns = vulns
	return filtered
}
