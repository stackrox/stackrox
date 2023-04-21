package common

// ResourceCountByCVESeverity provides functionality to retrieve the count of resources affected by cve distributed over severity.
type ResourceCountByCVESeverity interface {
	GetCriticalSeverityCount() ResourceCountByFixability
	GetImportantSeverityCount() ResourceCountByFixability
	GetModerateSeverityCount() ResourceCountByFixability
	GetLowSeverityCount() ResourceCountByFixability
}

// ResourceCountByFixability provides functionality to retrieve the count of resources affected by cve distributed over fixable property.
type ResourceCountByFixability interface {
	GetTotal() int
	GetFixable() int
}
