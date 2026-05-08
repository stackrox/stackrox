package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
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
	deploymentsQuery := search.EmptyQuery()
	var err error
	if q.collection != nil {
		deploymentsQuery, err = q.collectionQueryResolver.ResolveCollectionQuery(ctx, q.collection)
	} else if q.entityScope != nil {
		deploymentsQuery, err = q.buildEntityScopeQuery()
	}
	if err != nil {
		return nil, err
	}
	scopeQuery, err := q.buildAccessScopeQuery(clusters, namespaces)
	if err != nil {
		return nil, err
	}
	deploymentsQuery = search.ConjunctionQuery(deploymentsQuery, scopeQuery, deploymentDataStore.ActiveDeploymentsQuery())

	cveQuery, err := q.buildCVEAttributesQuery()
	if err != nil {
		return nil, err
	}
	return &ReportQuery{
		CveFieldsQuery:   cveQuery,
		DeploymentsQuery: deploymentsQuery,
	}, nil
}

// buildLegacyFilterQuery() adds severity, fixability filters for collection scoped reports
func (q *queryBuilder) buildLegacyFilterQuery() []string {

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
	return conjuncts
}

func (q *queryBuilder) buildCVEAttributesQuery() (string, error) {

	vulnReportFilters := q.vulnFilters
	var conjuncts []string

	if q.collection != nil {
		// for collections only add fixability, severity filters for CVE
		conjuncts = q.buildLegacyFilterQuery()
	} else if q.entityScope != nil {
		// for entity scoped reports add all the search filters from query string
		conjuncts = append(conjuncts, q.vulnFilters.GetQuery())
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

// buildEntityScopeQuery uses entity scope object to build v1 query
func (q *queryBuilder) buildEntityScopeQuery() (*v1.Query, error) {
	rules := q.entityScope.GetRules()
	if len(rules) == 0 {
		return search.EmptyQuery(), nil
	}

	var conjuncts []*v1.Query
	for _, rule := range rules {
		if len(rule.GetValues()) == 0 {
			continue
		}

		fieldLabel, err := entityScopeRuleToFieldLabel(rule)
		if err != nil {
			return nil, err
		}
		isMapField := fieldLabel == search.DeploymentLabel ||
			fieldLabel == search.NamespaceLabel ||
			fieldLabel == search.ClusterLabel ||
			fieldLabel == search.DeploymentAnnotation ||
			fieldLabel == search.NamespaceAnnotation

		if isMapField {
			mapQueries := make([]*v1.Query, 0, len(rule.GetValues()))
			for _, rv := range rule.GetValues() {
				val := rv.GetValue()
				key, value := splitLabelValue(val)
				mapQueries = append(mapQueries,
					search.NewQueryBuilder().AddMapQuery(fieldLabel, key, value).ProtoQuery())
			}
			conjuncts = append(conjuncts, search.DisjunctionQuery(mapQueries...))
		} else {
			var ruleQueries []*v1.Query
			for _, rv := range rule.GetValues() {
				val := rv.GetValue()
				if rv.GetMatchType() == storage.MatchType_REGEX {
					val = search.RegexPrefix + val
					ruleQueries = append(ruleQueries,
						search.NewQueryBuilder().AddStrings(fieldLabel, val).ProtoQuery())
				} else {
					ruleQueries = append(ruleQueries,
						search.NewQueryBuilder().AddExactMatches(fieldLabel, val).ProtoQuery())
				}
			}
			conjuncts = append(conjuncts, search.DisjunctionQuery(ruleQueries...))
		}
	}

	if len(conjuncts) == 0 {
		return search.EmptyQuery(), nil
	}
	return search.ConjunctionQuery(conjuncts...), nil
}

// entityScopeRuleToFieldLabel returns search filter for given entity field pair
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
	return "", errors.Errorf("Unsupported entity/field combination %s/%s", rule.GetEntity(), rule.GetField())
}

func filterVulnsByFirstOccurrenceTime(vulnReportFilters *storage.VulnerabilityReportFilters) bool {
	return vulnReportFilters.GetSinceLastSentScheduledReport() || vulnReportFilters.GetSinceStartDate() != nil
}

// split map field values like namespace labels(key=val) to key,val pair
func splitLabelValue(labelVal string) (string, string) {
	parts := strings.SplitN(labelVal, "=", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("%q", parts[0]), fmt.Sprintf("%q", parts[1])
	}
	return fmt.Sprintf("%q", labelVal), fmt.Sprintf("%q", "")
}
