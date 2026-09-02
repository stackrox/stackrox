package node

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var testClusters = []effectiveaccessscope.Cluster{
	&storage.Cluster{
		Id:   uuid.NewV4().String(),
		Name: "prod-us",
	},
	&storage.Cluster{
		Id:   uuid.NewV4().String(),
		Name: "prod-eu",
	},
}

func TestNodeEntityScopeRuleToFieldLabel(t *testing.T) {
	testCases := map[string]struct {
		rule     *storage.EntityScopeRule
		expected search.FieldLabel
		hasError bool
	}{
		"Cluster name returns search.Cluster": {
			rule: &storage.EntityScopeRule{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_NAME,
			},
			expected: search.Cluster,
		},
		"Cluster label returns search.ClusterLabel": {
			rule: &storage.EntityScopeRule{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_LABEL,
			},
			expected: search.ClusterLabel,
		},
		"Namespace entity is not allowed": {
			rule: &storage.EntityScopeRule{
				Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
				Field:  storage.EntityField_FIELD_NAME,
			},
			hasError: true,
		},
		"Deployment entity is not allowed": {
			rule: &storage.EntityScopeRule{
				Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
				Field:  storage.EntityField_FIELD_NAME,
			},
			hasError: true,
		},
		"Cluster annotation is not allowed": {
			rule: &storage.EntityScopeRule{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_ANNOTATION,
			},
			hasError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := nodeEntityScopeRuleToFieldLabel(tc.rule)
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildEntityScopeQuery(t *testing.T) {
	testCases := map[string]struct {
		scope    *storage.EntityScope
		expected *v1.Query
		hasError bool
	}{
		"Empty rules returns empty query": {
			scope:    &storage.EntityScope{},
			expected: search.EmptyQuery(),
		},
		"Single cluster name rule": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod-us", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected: search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod-us").ProtoQuery(),
		},
		"Multiple cluster name values are ORed": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod-us", MatchType: storage.MatchType_EXACT},
							{Value: "prod-eu", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod-us").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod-eu").ProtoQuery(),
			),
		},
		"Cluster label rule uses map query": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_LABEL,
						Values: []*storage.RuleValue{
							{Value: "env=prod", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected: search.NewQueryBuilder().AddMapQuery(search.ClusterLabel, `"env"`, `"prod"`).ProtoQuery(),
		},
		"Regex match type": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod-.*", MatchType: storage.MatchType_REGEX},
						},
					},
				},
			},
			expected: search.NewQueryBuilder().AddRegexes(search.Cluster, "prod-.*").ProtoQuery(),
		},
		"Multiple rules are ANDed": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod-us", MatchType: storage.MatchType_EXACT},
						},
					},
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_LABEL,
						Values: []*storage.RuleValue{
							{Value: "tier=premium", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod-us").ProtoQuery(),
				search.NewQueryBuilder().AddMapQuery(search.ClusterLabel, `"tier"`, `"premium"`).ProtoQuery(),
			),
		},
		"Rule with empty values is skipped": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{},
					},
				},
			},
			expected: search.EmptyQuery(),
		},
		"Namespace rule returns error": {
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "default", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			hasError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			qb := &queryBuilder{entityScope: tc.scope}
			result, err := qb.buildEntityScopeQuery()
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildQuery(t *testing.T) {
	testCases := map[string]struct {
		entityScope *storage.EntityScope
		filters     *storage.NodeVulnerabilityReportFilters
		hasError    bool
	}{
		"View-based (nil entity scope) builds without error": {
			entityScope: nil,
			filters:     &storage.NodeVulnerabilityReportFilters{},
		},
		"Config-based with entity scope builds without error": {
			entityScope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod-us", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			filters: &storage.NodeVulnerabilityReportFilters{},
		},
		"Config-based with query filter": {
			entityScope: &storage.EntityScope{},
			filters: &storage.NodeVulnerabilityReportFilters{
				Query: "CVE:CVE-2021-1234",
			},
		},
		"Invalid entity scope returns error": {
			entityScope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "default", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			filters:  &storage.NodeVulnerabilityReportFilters{},
			hasError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			qb := newQueryBuilder(tc.entityScope, tc.filters)
			result, err := qb.buildQuery(testClusters, nil)
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}
