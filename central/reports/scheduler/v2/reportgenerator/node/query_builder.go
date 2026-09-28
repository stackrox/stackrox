package node

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

type queryBuilder struct {
	filters     *storage.NodeVulnerabilityReportFilters
	entityScope *storage.EntityScope
}

func newQueryBuilder(entityScope *storage.EntityScope, filters *storage.NodeVulnerabilityReportFilters) *queryBuilder {
	return &queryBuilder{
		filters:     filters,
		entityScope: entityScope,
	}
}

func (q *queryBuilder) buildQuery(clusters []effectiveaccessscope.Cluster, namespaces []effectiveaccessscope.Namespace) (*v1.Query, error) {
	var conjuncts []*v1.Query

	if q.entityScope != nil {
		scopeQuery, err := q.buildEntityScopeQuery()
		if err != nil {
			return nil, err
		}
		conjuncts = append(conjuncts, scopeQuery)
	}

	accessScopeQuery, err := common.BuildClusterOnlyAccessScopeQuery(q.filters.GetAccessScopeRules(), clusters, namespaces)
	if err != nil {
		return nil, err
	}
	conjuncts = append(conjuncts, accessScopeQuery)

	cveFilterQuery, err := search.ParseQuery(q.filters.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	conjuncts = append(conjuncts, cveFilterQuery)

	return search.ConjunctionQuery(conjuncts...), nil
}

func (q *queryBuilder) buildEntityScopeQuery() (*v1.Query, error) {
	var conjuncts []*v1.Query
	for _, rule := range q.entityScope.GetRules() {
		if len(rule.GetValues()) == 0 {
			continue
		}
		fieldLabel, err := nodeEntityScopeRuleToFieldLabel(rule)
		if err != nil {
			return nil, err
		}
		if fieldLabel == search.ClusterLabel {
			conjuncts = append(conjuncts, common.MapFieldQuery(fieldLabel, rule))
		} else {
			conjuncts = append(conjuncts, common.ScalarFieldQuery(fieldLabel, rule))
		}
	}

	if len(conjuncts) == 0 {
		return search.EmptyQuery(), nil
	}
	return search.ConjunctionQuery(conjuncts...), nil
}

func nodeEntityScopeRuleToFieldLabel(rule *storage.EntityScopeRule) (search.FieldLabel, error) {
	if rule.GetEntity() != storage.EntityType_ENTITY_TYPE_CLUSTER {
		return "", errors.Errorf("unsupported entity type %s for node reports; only ENTITY_TYPE_CLUSTER is allowed", rule.GetEntity())
	}
	switch rule.GetField() {
	case storage.EntityField_FIELD_NAME:
		return search.Cluster, nil
	case storage.EntityField_FIELD_LABEL:
		return search.ClusterLabel, nil
	}
	return "", errors.Errorf("unsupported field %s for cluster entity scope", rule.GetField())
}
