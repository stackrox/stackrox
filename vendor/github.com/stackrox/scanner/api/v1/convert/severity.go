package convert

import "github.com/stackrox/scanner/database"

// Severity is the uniform severity returned through the API
type Severity string

// Severity settings for vulnerabilities
const (
	UnknownSeverity   Severity = "Unknown"
	LowSeverity       Severity = "Low"
	ModerateSeverity  Severity = "Moderate"
	ImportantSeverity Severity = "Important"
	CriticalSeverity  Severity = "Critical"
)

// DatabaseSeverityToSeverity converts a database.Severity into a Severity.
func DatabaseSeverityToSeverity(severity database.Severity) Severity {
	switch severity {
	case database.UnknownSeverity:
		return UnknownSeverity
	case database.NegligibleSeverity, database.LowSeverity:
		return LowSeverity
	case database.MediumSeverity:
		return ModerateSeverity
	case database.HighSeverity:
		return ImportantSeverity
	case database.CriticalSeverity, database.Defcon1Severity:
		return CriticalSeverity
	}
	return LowSeverity
}
