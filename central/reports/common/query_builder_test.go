package common

import (
	"github.com/stackrox/rox/generated/storage"
)

var clusters = []*storage.Cluster{
	{
		Id:   "cluster1",
		Name: "remote",
	},
	{
		Id:   "cluster2",
		Name: "secured",
	},
}

var namespaces = []*storage.NamespaceMetadata{
	remoteNS,
	securedNS,
}

var remoteNS = &storage.NamespaceMetadata{
	Id:          "namespace1",
	Name:        "ns1",
	ClusterId:   "cluster1",
	ClusterName: "remote",
}

var securedNS = &storage.NamespaceMetadata{
	Id:          "namespace2",
	Name:        "ns2",
	ClusterId:   "cluster2",
	ClusterName: "secured",
}

var vulnFilters = &storage.VulnerabilityReportFilters{
	Fixability:      storage.VulnerabilityReportFilters_FIXABLE,
	SinceLastReport: false,
	Severities: []storage.VulnerabilitySeverity{
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
	},
}
