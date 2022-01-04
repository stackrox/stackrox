package scheduler

import (
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

type reportQuery struct {
	cveFieldsQuery string
	scopeQueries   []string
}

type queryBuilder struct {
	clusters              []*storage.Cluster
	namespaces            []*storage.NamespaceMetadata
	scope                 *storage.SimpleAccessScope
	vulnFilters           *storage.VulnerabilityReportFilters
	lastSuccessfulRunTime time.Time
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilder(clusters []*storage.Cluster,
	namespaces []*storage.NamespaceMetadata, scope *storage.SimpleAccessScope,
	vulnFilters *storage.VulnerabilityReportFilters, lastSuccessfulRunTime time.Time) *queryBuilder {
	return &queryBuilder{
		clusters:              clusters,
		namespaces:            namespaces,
		scope:                 scope,
		vulnFilters:           vulnFilters,
		lastSuccessfulRunTime: lastSuccessfulRunTime,
	}
}

// BuildQuery builds scope and cve filtering queries for vuln reporting
func (q *queryBuilder) BuildQuery() (*reportQuery, error) {
	scopeQueries, err := q.buildScopeQueries()
	if err != nil {
		return nil, err
	}
	cveQuery, err := q.buildCVEAttributesQuery()
	if err != nil {
		return nil, err
	}
	return &reportQuery{
		cveQuery,
		scopeQueries,
	}, nil
}

func (q *queryBuilder) buildCVEAttributesQuery() (string, error) {
	vulnReportFilters := q.vulnFilters
	var conjuncts []*v1.Query

	switch vulnReportFilters.GetFixability() {
	case storage.VulnerabilityReportFilters_BOTH:
		break
	case storage.VulnerabilityReportFilters_FIXABLE:
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddStrings(search.Fixable, "true").ProtoQuery())
	case storage.VulnerabilityReportFilters_NOT_FIXABLE:
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddStrings(search.Fixable, "false").ProtoQuery())
	}

	sevQueries := make([]*v1.Query, len(vulnReportFilters.GetSeverities()))
	for _, severity := range vulnReportFilters.GetSeverities() {
		sevQueries = append(sevQueries, search.NewQueryBuilder().AddStrings(search.Severity, severity.String()).ProtoQuery())
	}
	if len(sevQueries) > 0 {
		conjuncts = append(conjuncts, search.DisjunctionQuery(sevQueries...))
	}

	if vulnReportFilters.SinceLastReport {
		reportLastSuccessfulRunTs := fmt.Sprintf("<%s", q.lastSuccessfulRunTime)
		conjuncts = append(conjuncts, search.NewQueryBuilder().
			AddGenericTypeLinkedFields([]search.FieldLabel{search.FirstImageOccurrenceTimestamp},
				[]interface{}{reportLastSuccessfulRunTs}).ProtoQuery())
	}
	return search.ConjunctionQuery(conjuncts...).String(), nil
}

func (q *queryBuilder) buildScopeQueries() ([]string, error) {
	tree, err := effectiveaccessscope.ComputeEffectiveAccessScope(q.scope.GetRules(), q.clusters, q.namespaces, v1.ComputeEffectiveAccessScopeRequest_STANDARD)
	if err != nil {
		return nil, err
	}
	return tree.Compactify().ToScopeQueries(), nil
}
