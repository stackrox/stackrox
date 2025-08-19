package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

// ReportQuery encapsulates the cve specific fields query, and the resource scope
// queries to be used in a report generation run
type ReportQueryViewBased struct {
	CveFieldsQuery         string
	DeploymentsScopedQuery *v1.Query
}

type queryBuilderViewBased struct {
	vulnFilters *storage.ViewBasedVulnerabilityReportFilters
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilderViewBased(vulnFilters *storage.ViewBasedVulnerabilityReportFilters) *queryBuilderViewBased {
	return &queryBuilderViewBased{
		vulnFilters: vulnFilters,
	}
}

// BuildQueryViewBased builds scope and cve filtering queries for view-based vuln reporting
func (q *queryBuilderViewBased) BuildQueryViewBased(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*ReportQueryViewBased, error) {
	// For view-based reports, we don't need access scope filtering since the user's query
	// should already contain the appropriate filters
	_, err := q.buildAccessScopeQueryViewBased(clusters, namespaces)
	if err != nil {
		return nil, err
	}

	cveQuery := q.buildCVEAttributesQueryViewBased()

	// For view-based reports, we need to ensure that deployment information is included
	// when the user requests deployed images. We'll use an empty query for the deployment scope
	// since the user's query should already contain the appropriate filters.
	deploymentsQuery := search.EmptyQuery()

	return &ReportQueryViewBased{
		CveFieldsQuery:         cveQuery,
		DeploymentsScopedQuery: deploymentsQuery,
	}, nil
}

func (q *queryBuilderViewBased) buildCVEAttributesQueryViewBased() string {
	vulnReportFilters := q.vulnFilters
	return vulnReportFilters.GetQuery()
}

func (q *queryBuilderViewBased) buildAccessScopeQueryViewBased(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*v1.Query, error) {
	accessScopeRules := q.vulnFilters.GetAccessScopeRules()
	if accessScopeRules == nil {
		// Old(v1) report configurations would have nil access scope rules.
		// For backward compatibility, nil access scope would mean access to all clusters and namespaces.
		// To deny access to all clusters and namespaces, the accessScopeRules should be empty.
		return search.EmptyQuery(), nil
	}
	var scopeTree *effectiveaccessscope.ScopeTree
	for _, rules := range accessScopeRules {
		sct, err := effectiveaccessscope.ComputeEffectiveAccessScope(rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
		if err != nil {
			return nil, err
		}
		if scopeTree == nil {
			scopeTree = sct
		} else {
			scopeTree.Merge(sct)
		}
	}
	scopeQuery, err := sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
	if err != nil {
		return nil, err
	}
	if scopeQuery == nil {
		return search.EmptyQuery(), nil
	}
	return scopeQuery, nil
}
