package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	entityScope             *storage.EntityScope
}

// NewVulnReportQueryBuilder builds a query builder to build scope and cve filtering queries for vuln reporting
func NewVulnReportQueryBuilder(collection *storage.ResourceCollection, entityScope *storage.EntityScope, vulnFilters *storage.VulnerabilityReportFilters,
	collectionQueryRes collectionDataStore.QueryResolver, dataStartTime time.Time) *queryBuilder {
	return &queryBuilder{
		collection:              collection,
		entityScope:             entityScope,
		vulnFilters:             vulnFilters,
		collectionQueryResolver: collectionQueryRes,
		dataStartTime:           dataStartTime,
	}

}

// BuildQuery builds scope and cve filtering queries for vuln reporting
func (q *queryBuilder) BuildQuery(
	ctx context.Context,
	clusters []effectiveaccessscope.Cluster,
	namespaces []effectiveaccessscope.Namespace,
) (*ReportQuery, error) {
	deploymentsQuery := search.MatchNoneQuery()
	var err error
	if q.collection != nil {
		deploymentsQuery, err = q.collectionQueryResolver.ResolveCollectionQuery(ctx, q.collection)
		if err != nil {
			return nil, err
		}
	} else {

		entityScopeQueryString, err := q.buildEntityScopeQueryString()
		if err != nil {
			return nil, err
		}
		deploymentsQuery, err = search.ParseQuery(entityScopeQueryString, search.MatchAllIfEmpty())
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

func (q *queryBuilder) buildEntityScopeQueryString() (string, error) {
	rules := q.entityScope.GetRules()
	if len(rules) == 0 {
		return "", nil
	}

	var conjuncts []string
	for _, rule := range rules {
		fieldLabel, err := entityScopeRuleToFieldLabel(rule)
		if err != nil {
			return "", err
		}
		isLabel := fieldLabel == search.DeploymentLabel ||
			fieldLabel == search.NamespaceLabel ||
			fieldLabel == search.ClusterLabel

		values := make([]string, 0, len(rule.GetValues()))
		for _, rv := range rule.GetValues() {
			val := rv.GetValue()
			if rv.GetMatchType() == storage.MatchType_REGEX {
				val = search.RegexPrefix + val
			}
			values = append(values, val)
		}

		if len(values) == 0 {
			continue
		}

		var qb *search.QueryBuilder
		if isLabel {
			for _, v := range values {
				key, value := splitLabelValue(v)
				qb = search.NewQueryBuilder().AddMapQuery(fieldLabel, key, value)
				conjuncts = append(conjuncts, qb.Query())
			}
		} else {
			qb = search.NewQueryBuilder().AddExactMatches(fieldLabel, values...)
			conjuncts = append(conjuncts, qb.Query())
		}
	}

	return strings.Join(conjuncts, "+"), nil
}

func entityScopeRuleToFieldLabel(rule *storage.EntityScopeRule) (search.FieldLabel, error) {
	switch rule.GetEntity() {
	case storage.EntityType_ENTITY_TYPE_DEPLOYMENT:
		switch rule.GetField() {
		case storage.EntityField_FIELD_NAME:
			return search.DeploymentName, nil
		case storage.EntityField_FIELD_LABEL:
			return search.DeploymentLabel, nil
		case storage.EntityField_FIELD_ANNOTATION:
			return search.DeploymentAnnotation, nil
		}
	case storage.EntityType_ENTITY_TYPE_NAMESPACE:
		switch rule.GetField() {
		case storage.EntityField_FIELD_NAME:
			return search.Namespace, nil
		case storage.EntityField_FIELD_LABEL:
			return search.NamespaceLabel, nil
		case storage.EntityField_FIELD_ANNOTATION:
			return search.NamespaceAnnotation, nil
		}
	case storage.EntityType_ENTITY_TYPE_CLUSTER:
		switch rule.GetField() {
		case storage.EntityField_FIELD_NAME:
			return search.Cluster, nil
		case storage.EntityField_FIELD_LABEL:
			return search.ClusterLabel, nil
		}
	}
	return "", fmt.Errorf("unsupported entity/field combination: %s/%s", rule.GetEntity(), rule.GetField())
}

func splitLabelValue(labelVal string) (string, string) {
	parts := strings.SplitN(labelVal, "=", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("%q", parts[0]), fmt.Sprintf("%q", parts[1])
	}
	return fmt.Sprintf("%q", labelVal), fmt.Sprintf("%q", "")
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

func (q *queryBuilder) buildAccessScopeQuery(
	clusters []effectiveaccessscope.Cluster,
	namespaces []effectiveaccessscope.Namespace,
) (*v1.Query, error) {
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

func filterVulnsByFirstOccurrenceTime(vulnReportFilters *storage.VulnerabilityReportFilters) bool {
	return vulnReportFilters.GetSinceLastSentScheduledReport() || vulnReportFilters.GetSinceStartDate() != nil
}
