package cvss

import (
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/pkg/types"
)

// SeverityFromCVSS converts the CVSS Score (0.0 - 10.0) into a
// database.Severity following the qualitative rating scale available in the
// CVSS v3.0 specification (https://www.first.org/cvss/specification-document),
// Table 14.
//
// The Negligible level is set for CVSS scores between [0, 1), replacing the
// specified None level, originally used for a score of 0.
func SeverityFromCVSS(meta *types.Metadata) database.Severity {
	score := meta.CVSSv3.Score
	if score == 0 {
		score = meta.CVSSv2.Score
	}
	switch {
	case score < 1.0:
		return database.NegligibleSeverity
	case score < 4.0:
		return database.LowSeverity
	case score < 7.0:
		return database.MediumSeverity
	case score < 9.0:
		return database.HighSeverity
	case score <= 10.0:
		return database.CriticalSeverity
	}
	return database.UnknownSeverity
}
