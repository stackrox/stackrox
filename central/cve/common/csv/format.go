package csv

import "fmt"

// FormatCVSSScore formats a CVSS score and its version for vulnerability exports.
func FormatCVSSScore(score *float64, version string) string {
	if score == nil {
		return "Not Available"
	}
	return fmt.Sprintf("%.2f (%s)", *score, version)
}
