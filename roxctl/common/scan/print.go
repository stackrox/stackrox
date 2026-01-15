package scan

import "github.com/stackrox/rox/roxctl/common/logger"

// PrintCVESummary print summary of amount of CVEs found
func PrintCVESummary(image string, cveSummary map[string]int, out logger.Logger) {
	out.PrintfLn("Scan results for image: %s", image)
	out.PrintfLn("(%s: %d, %s: %d, %s: %d, %s: %d, %s: %d, %s: %d)\n",
		totalComponentsMapKey, cveSummary[totalComponentsMapKey],
		totalVulnerabilitiesMapKey, cveSummary[totalVulnerabilitiesMapKey],
		LowCVESeverity, cveSummary[LowCVESeverity.String()],
		ModerateCVESeverity, cveSummary[ModerateCVESeverity.String()],
		ImportantCVESeverity, cveSummary[ImportantCVESeverity.String()],
		CriticalCVESeverity, cveSummary[CriticalCVESeverity.String()])
}

// PrintCVEWarning print warning with amount of CVEs found in components
func PrintCVEWarning(numOfVulns int, numOfComponents int, out logger.Logger) {
	if numOfVulns != 0 {
		out.WarnfLn("A total of %d unique vulnerabilities were found in %d components",
			numOfVulns, numOfComponents)
	}
}
