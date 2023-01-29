package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

// ReportQuery encapsulates the cve specific fields query, and the resource scope
// queries to be used in a report generation run
type ReportQuery struct {
	CveFieldsQuery string
	// ScopeQueries are used when scoping vuln report using an access-scope
	ScopeQueries []string
	// DeploymentsQuery is used when scoping vuln report using a resource-collection
	DeploymentsQuery *v1.Query
}

type queryBuilder struct {
	clusters                []*storage.Cluster
	namespaces              []*storage.NamespaceMetadata
	scope                   *storage.SimpleAccessScope
	vulnFilters             *storage.VulnerabilityReportFilters
	collection              *storage.ResourceCollection
	collectionQueryResolver collectionDataStore.QueryResolver
	lastSuccessfulRunTime   time.Time
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilder(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata, scope *storage.SimpleAccessScope, collection *storage.ResourceCollection,
	vulnFilters *storage.VulnerabilityReportFilters, collectionQueryRes collectionDataStore.QueryResolver,
	lastSuccessfulRunTime time.Time) *queryBuilder {
	return &queryBuilder{
		clusters:                clusters,
		namespaces:              namespaces,
		scope:                   scope,
		vulnFilters:             vulnFilters,
		collection:              collection,
		collectionQueryResolver: collectionQueryRes,
		lastSuccessfulRunTime:   lastSuccessfulRunTime,
	}
}

// BuildQuery builds scope and cve filtering queries for vuln reporting
func (q *queryBuilder) BuildQuery(ctx context.Context) (*ReportQuery, error) {
	var err error
	var deploymentsQuery *v1.Query
	var scopeQueries []string
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		deploymentsQuery, err = q.collectionQueryResolver.ResolveCollectionQuery(ctx, q.collection)
	} else {
		scopeQueries, err = q.buildScopeQueries()
	}

	if err != nil {
		return nil, err
	}
	cveQuery, err := q.buildCVEAttributesQuery()
	if err != nil {
		return nil, err
	}
	return &ReportQuery{
		cveQuery,
		scopeQueries,
		deploymentsQuery,
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

	if vulnReportFilters.SinceLastReport {
		reportLastSuccessfulRunTs := fmt.Sprintf(">=%s", q.lastSuccessfulRunTime.Format("01/02/2006 3:04:05 PM MST"))
		tsQ := search.NewQueryBuilder().AddStrings(search.FirstImageOccurrenceTimestamp, reportLastSuccessfulRunTs).Query()
		conjuncts = append(conjuncts, tsQ)
	}
	return strings.Join(conjuncts, "+"), nil
}

func (q *queryBuilder) buildScopeQueries() ([]string, error) {
	tree, err := effectiveaccessscope.ComputeEffectiveAccessScope(q.scope.GetRules(), q.clusters, q.namespaces, v1.ComputeEffectiveAccessScopeRequest_STANDARD)
	if err != nil {
		return nil, err
	}
	return tree.Compactify().ToScopeQueries(), nil
}
