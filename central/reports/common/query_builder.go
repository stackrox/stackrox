package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

// ReportQuery encapsulates the cve specific fields query, and the resource scope
// queries to be used in a report generation run
type ReportQuery struct {
	CveFieldsQuery string
	// DeploymentsQuery is used when scoping vuln report using a resource-collection
	DeploymentsQuery *v1.Query
}

type queryBuilder struct {
	vulnFilters             *storage.VulnerabilityReportFilters
	collection              *storage.ResourceCollection
	collectionQueryResolver collectionDataStore.QueryResolver
	dataStartTime           time.Time
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilder(collection *storage.ResourceCollection, vulnFilters *storage.VulnerabilityReportFilters,
	collectionQueryRes collectionDataStore.QueryResolver, dataStartTime time.Time) *queryBuilder {
	return &queryBuilder{
		vulnFilters:             vulnFilters,
		collection:              collection,
		collectionQueryResolver: collectionQueryRes,
		dataStartTime:           dataStartTime,
	}
}

// BuildQuery builds scope and cve filtering queries for vuln reporting
func (q *queryBuilder) BuildQuery(ctx context.Context, clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata) (*ReportQuery, error) {
	deploymentsQuery, err := q.collectionQueryResolver.ResolveCollectionQuery(ctx, q.collection)
	if err != nil {
		return nil, err
	}
	scopeQuery, err := q.buildAccessScopeQuery(clusters, namespaces)
	if err != nil {
		return nil, err
	}
	deploymentsQuery = search.ConjunctionQuery(deploymentsQuery, scopeQuery)

	cveQuery, err := q.buildCVEAttributesQuery()
	if err != nil {
		return nil, err
	}
	return &ReportQuery{
		CveFieldsQuery:   cveQuery,
		DeploymentsQuery: deploymentsQuery,
	}, nil
}

func (q *queryBuilder) buildCVEAttributesQuery() (string, error) {
	vulnReportFilters := q.vulnFilters
	var conjuncts []string

	switch vulnReportFilters.GetFixability() {
	case storage.VulnerabilityReportFilters_BOTH:
		break
	case storage.VulnerabilityReportFilters_FIXABLE:
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddBools(search.Fixable, true).Query())
	case storage.VulnerabilityReportFilters_NOT_FIXABLE:
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddBools(search.Fixable, false).Query())
	}

	severities := make([]string, 0, len(vulnReportFilters.GetSeverities()))
	for _, severity := range vulnReportFilters.GetSeverities() {
		severities = append(severities, severity.String())
	}
	if len(severities) > 0 {
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddExactMatches(search.Severity, severities...).Query())
	}

	if filterVulnsByFirstOccurrenceTime(vulnReportFilters) {
		startTimeStr := fmt.Sprintf(">=%s", q.dataStartTime.Format("01/02/2006 3:04:05 PM MST"))
		tsQ := search.NewQueryBuilder().AddStrings(search.FirstImageOccurrenceTimestamp, startTimeStr).Query()
		conjuncts = append(conjuncts, tsQ)
	}

	return strings.Join(conjuncts, "+"), nil
}

func (q *queryBuilder) buildAccessScopeQuery(clusters []*storage.Cluster,
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
	scopeQuery, err := sac.BuildNonVerboseClusterNamespaceLevelSACQueryFilter(scopeTree)
	if err != nil {
		return nil, err
	}
	if scopeQuery == nil {
		return search.EmptyQuery(), nil
	}
	return scopeQuery, nil
}

func filterVulnsByFirstOccurrenceTime(vulnReportFilters *storage.VulnerabilityReportFilters) bool {
	if !features.VulnReportingEnhancements.Enabled() {
		return vulnReportFilters.SinceLastReport
	}
	return vulnReportFilters.GetSinceLastSentScheduledReport() || vulnReportFilters.GetSinceStartDate() != nil
}
