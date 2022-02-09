package common

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
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

var accessScope = &storage.SimpleAccessScope{
	Id:   "scope1",
	Name: "test scope",
	Rules: &storage.SimpleAccessScope_Rules{
		IncludedClusters: []string{},
		IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
			{
				NamespaceName: "ns1",
				ClusterName:   "remote",
			},
			{
				NamespaceName: "ns2",
				ClusterName:   "secured",
			},
		},
	},
}

var vulnFilters = &storage.VulnerabilityReportFilters{
	Fixability:      storage.VulnerabilityReportFilters_FIXABLE,
	SinceLastReport: false,
	Severities: []storage.VulnerabilitySeverity{
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
	},
}

func TestBuildQuery(t *testing.T) {
	qb := NewVulnReportQueryBuilder(clusters, namespaces, accessScope, vulnFilters, time.Now())
	rq, err := qb.BuildQuery()
	assert.NoError(t, err)

	assert.ElementsMatch(t, []string{"Cluster:remote+Namespace:ns1", "Cluster:secured+Namespace:ns2"}, rq.ScopeQueries)
	assert.Equal(t, "Fixable:true+Severity:CRITICAL_VULNERABILITY_SEVERITY,IMPORTANT_VULNERABILITY_SEVERITY", rq.CveFieldsQuery)
}
