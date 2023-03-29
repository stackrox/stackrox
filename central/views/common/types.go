package common

// ResourceCountByCVESeverity is the count of resources affected by cve distributed over severity.
type ResourceCountByCVESeverity interface {
	GetCriticalSeverityCount() int
	GetImportantSeverityCount() int
	GetModerateSeverityCount() int
	GetLowSeverityCount() int
}
