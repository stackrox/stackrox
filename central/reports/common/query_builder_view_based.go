package common

import (
	"time"

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
	vulnFilters   *storage.ViewBasedVulnerabilityReportFilters
	dataStartTime time.Time
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilderViewBased(vulnFilters *storage.ViewBasedVulnerabilityReportFilters, dataStartTime time.Time) *queryBuilderViewBased {
	return &queryBuilderViewBased{
		vulnFilters:   vulnFilters,
		dataStartTime: dataStartTime,
	}
}

// BuildQuery builds scope and cve filtering queries for vuln reporting
func (q *queryBuilderViewBased) BuildQueryViewBased(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*ReportQueryViewBased, error) {
	scopeQuery, err := q.buildAccessScopeQueryViewBased(clusters, namespaces)
	if err != nil {
		return nil, err
	}

	cveQuery := q.buildCVEAttributesQueryViewBased()
	if err != nil {
		return nil, err
	}
	return &ReportQueryViewBased{
		CveFieldsQuery:         cveQuery,
		DeploymentsScopedQuery: scopeQuery,
	}, nil
}

func (q *queryBuilderViewBased) buildCVEAttributesQueryViewBased() string {
	vulnReportFilters := q.vulnFilters
	return vulnReportFilters.GetQuery()
}

func (q *queryBuilderViewBased) buildAccessScopeQueryViewBased(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*v1.Query, error) {
	accessScopeRules := q.vulnFilters.GetAccessScopeRules()
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

func filterVulnsByFirstOccurrenceTimeViewBased(vulnReportFilters *storage.VulnerabilityReportFilters) bool {
	return vulnReportFilters.GetSinceLastSentScheduledReport() || vulnReportFilters.GetSinceStartDate() != nil
}
