package cvss

// Severity represents cve severity by cvss v3 (preferred) or cvss v2
type Severity int32

const (
	// UNKNOWN represents cve severity is unset
	UNKNOWN Severity = iota
	// LOW represents low cve severity; cvss v2 (0.0-3.9), cvss v3 (0.0-3.9) NONE is also mapped to LOW
	LOW
	// MEDIUM represents medium cve severity; cvss v2 (4.0-6.9), cvss v3 (4.0-6.9)
	MEDIUM
	// HIGH represents high cve severity; cvss v2 (7.0-10.0), cvss v3 (7.0-8.9)
	HIGH
	// CRITICAL represents critical cve severity; cvss v2 (n/a), cvss v3 (9.0.-10.0)
	CRITICAL
)

func (s Severity) String() string {
	return [...]string{"UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL"}[s]
}

// ScoreToSeverity returns the top level cve severity for the cvss v3/v2 score
func ScoreToSeverity(score float32) Severity {
	switch {
	case score < 4.0:
		return LOW
	case score < 7.0:
		return MEDIUM
	case score < 9.0:
		return HIGH
	case score <= 10.0:
		return CRITICAL
	}
	return UNKNOWN
}
