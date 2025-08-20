package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ReportQueryViewBased encapsulates the cve specific fields query, and the resource scope
// queries to be used in a report generation run
type ReportQueryViewBased struct {
	CveFieldsQuery         string
	DeploymentsScopedQuery *v1.Query
}

type queryBuilderViewBased struct {
	vulnFilters *storage.ViewBasedVulnerabilityReportFilters
}

// NewVulnReportQueryBuilderViewBased builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilderViewBased(vulnFilters *storage.ViewBasedVulnerabilityReportFilters) *queryBuilderViewBased {
	return &queryBuilderViewBased{
		vulnFilters: vulnFilters,
	}
}

// BuildQueryViewBased builds scope and cve filtering queries for view-based vuln reporting
func (q *queryBuilderViewBased) BuildQueryViewBased(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*ReportQueryViewBased, error) {
	deploymentsScopedQuery, err := BuildAccessScopeQueryViewBased(q.vulnFilters.GetAccessScopeRules(), clusters, namespaces)
	if err != nil {
		return nil, err
	}

	cveQuery := q.buildCVEAttributesQueryViewBased()

	return &ReportQueryViewBased{
		CveFieldsQuery:         cveQuery,
		DeploymentsScopedQuery: deploymentsScopedQuery,
	}, nil
}

func (q *queryBuilderViewBased) buildCVEAttributesQueryViewBased() string {
	vulnReportFilters := q.vulnFilters
	return vulnReportFilters.GetQuery()
}
