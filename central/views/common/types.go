package common

// ResourceCountByCVESeverity provides functionality to retrieve the count of resources associated or affected by cve distributed over severity.
// The resource can be any resource associated with vulnerabilities, such as deployment, image, cve.
// For example:
// - if the resource is CVE, then the interface can provide functionality to return count of deployments, images, namespaces, etc. affected by the resource.
// - if the resource is deployment, then the interface can provide functionality to return count of CVEs and images.
type ResourceCountByCVESeverity interface {
	GetCriticalSeverityCount() ResourceCountByFixability
	GetImportantSeverityCount() ResourceCountByFixability
	GetModerateSeverityCount() ResourceCountByFixability
	GetLowSeverityCount() ResourceCountByFixability
}

// ResourceCountByFixability provides functionality to retrieve the count of resources affected by cve distributed over fixable property.
// The resource can be any resource associated with vulnerabilities, such as deployment, image, cve.
// For example:
// - if the resource is CVE, then the interface can provide functionality to return count of deployments, images, namespaces, etc. affected by the resource.
// - if the resource is deployment, then the interface can provide functionality to return count of total and fixable CVEs and total and fixable images.
type ResourceCountByFixability interface {
	GetTotal() int
	GetFixable() int
}
