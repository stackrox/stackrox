package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ReportQueryViewBased returns deployed images query and watched images query
type ReportQueryViewBased struct {
	DeployedImagesQuery *v1.Query
	WatchedImagesQuery  *v1.Query
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
	namespaces []*storage.NamespaceMetadata, watchedImages []string) (*ReportQueryViewBased, error) {
	deploymentsScopedQuery, err := BuildAccessScopeQuery(q.vulnFilters.GetAccessScopeRules(), clusters, namespaces)
	if err != nil {
		return nil, err
	}

	cveQuery := q.getViewBasedReportQueryString()
	cveFilterQuery, err := search.ParseQuery(cveQuery, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	deployedImagesQuery := search.ConjunctionQuery(deploymentsScopedQuery, cveFilterQuery)

	watchedImagesQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ImageName, watchedImages...).ProtoQuery(),
		cveFilterQuery)

	return &ReportQueryViewBased{
		deployedImagesQuery,
		watchedImagesQuery,
	}, nil
}

func (q *queryBuilderViewBased) getViewBasedReportQueryString() string {
	vulnReportFilters := q.vulnFilters
	return vulnReportFilters.GetQuery()
}
