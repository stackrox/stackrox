package common

import (
	"testing"

	rolePkg "github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	clusters = []effectiveaccessscope.Cluster{
		&storage.Cluster{
			Id:   uuid.NewV4().String(),
			Name: "remote",
		},
		&storage.Cluster{
			Id:   uuid.NewV4().String(),
			Name: "secured",
		},
	}

	namespaces = []effectiveaccessscope.Namespace{
		remoteNS,
		securedNS,
	}

	remoteNS = &storage.NamespaceMetadata{
		Id:          "namespace1",
		Name:        "ns1",
		ClusterId:   clusters[0].GetId(),
		ClusterName: "remote",
	}

	securedNS = &storage.NamespaceMetadata{
		Id:          "namespace2",
		Name:        "ns2",
		ClusterId:   clusters[1].GetId(),
		ClusterName: "secured",
	}
)

func getMatchNoneQuery() *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchNoneQuery{
					MatchNoneQuery: &v1.MatchNoneQuery{},
				},
			},
		},
	}
}

func TestBuildAccessScopeQuery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockID := mockIdentity.NewMockIdentity(mockCtrl)
	testCases := []struct {
		name          string
		identityGen   func() authn.Identity
		expectedQ     *v1.Query
		assertQueries func(t testing.TB, expected *v1.Query, actual *v1.Query)
	}{
		{
			name: "Identity has no roles",
			identityGen: func() authn.Identity {
				mockID.EXPECT().Roles().Return(nil).Times(1)
				return mockID
			},
			expectedQ:     getMatchNoneQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has nil access scope",
			identityGen: func() authn.Identity {
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(nil).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ:     getMatchNoneQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has exclude all access scope",
			identityGen: func() authn.Identity {
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(rolePkg.AccessScopeExcludeAll).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ:     getMatchNoneQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has include all access scope",
			identityGen: func() authn.Identity {
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(rolePkg.AccessScopeIncludeAll).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ:     search.EmptyQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has include all access scope among multiple access scopes",
			identityGen: func() authn.Identity {
				accessScope := &storage.SimpleAccessScope{
					Rules: &storage.SimpleAccessScope_Rules{
						IncludedClusters: []string{clusters[0].GetName()},
					},
				}
				mockRole1 := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole1.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockRole2 := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole2.EXPECT().GetAccessScope().Return(rolePkg.AccessScopeIncludeAll).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole1, mockRole2}).Times(1)
				return mockID
			},
			expectedQ:     search.EmptyQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has access scope with nil rules; access scope is not equal to AccessScopeIncludeAll system scope",
			identityGen: func() authn.Identity {
				accessScope := &storage.SimpleAccessScope{}
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ:     getMatchNoneQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Identity has access scope with rules",
			identityGen: func() authn.Identity {
				accessScope := &storage.SimpleAccessScope{
					Rules: &storage.SimpleAccessScope_Rules{
						IncludedClusters: []string{clusters[0].GetName()},
						IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
							{ClusterName: clusters[1].GetName(), NamespaceName: securedNS.GetName()},
						},
					},
				}
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusters[0].GetId()).ProtoQuery(),
				search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusters[1].GetId()).ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, securedNS.GetName()).ProtoQuery(),
				),
			),
			assertQueries: func(t testing.TB, expected *v1.Query, actual *v1.Query) {
				switch typedQ := actual.GetQuery().(type) {
				case *v1.Query_Disjunction:
					protoassert.ElementsMatch(t,
						expected.GetQuery().(*v1.Query_Disjunction).Disjunction.GetQueries(),
						typedQ.Disjunction.GetQueries())
				default:
					assert.Fail(t, "queries mismatch")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			identity := tc.identityGen()
			scopeRules := ExtractAccessScopeRules(identity)
			vulnReportFilters := &storage.VulnerabilityReportFilters{
				AccessScopeRules: scopeRules,
			}
			qBuilder := queryBuilder{vulnFilters: vulnReportFilters}
			scopeQuery, err := qBuilder.buildAccessScopeQuery(clusters, namespaces)
			assert.NoError(t, err)
			tc.assertQueries(t, tc.expectedQ, scopeQuery)
		})
	}
}

func assertByDirectComparison(t testing.TB, expected *v1.Query, actual *v1.Query) {
	protoassert.Equal(t, expected, actual)
}

func TestBuildEntityScopeQuery(t *testing.T) {
	testCases := []struct {
		name          string
		scope         *storage.EntityScope
		expected      *v1.Query
		assertQueries func(t testing.TB, expected *v1.Query, actual *v1.Query)
		hasError      bool
	}{
		{
			name:          "Empty rules returns empty query (match all)",
			scope:         &storage.EntityScope{},
			expected:      search.EmptyQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Namespace rule",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod", MatchType: storage.MatchType_EXACT},
							{Value: "staging", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected:      search.NewQueryBuilder().AddExactMatches(search.Namespace, "prod", "staging").ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Single deployment name rule",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "web-server", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected:      search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "web-server").ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Cluster name rule",
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
			expected:      search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod-us", "prod-eu").ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Multiple rules are ANDed",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "prod", MatchType: storage.MatchType_EXACT},
						},
					},
					{
						Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "backend", MatchType: storage.MatchType_EXACT},
							{Value: "frontend", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.Namespace, "prod").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "backend", "frontend").ProtoQuery(),
			),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Label rule uses map query",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_LABEL,
						Values: []*storage.RuleValue{
							{Value: "env=prod", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected:      search.NewQueryBuilder().AddMapQuery(search.NamespaceLabel, `"env"`, `"prod"`).ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Regex match type adds r/ prefix",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{
							{Value: "web-.*", MatchType: storage.MatchType_REGEX},
						},
					},
				},
			},
			expected:      search.NewQueryBuilder().AddStrings(search.DeploymentName, "r/web-.*").ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Rule with empty values is skipped",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
						Field:  storage.EntityField_FIELD_NAME,
						Values: []*storage.RuleValue{},
					},
				},
			},
			expected:      search.EmptyQuery(),
			assertQueries: assertByDirectComparison,
		},
		{
			name: "Unsupported entity/field returns error",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
						Field:  storage.EntityField_FIELD_ANNOTATION,
						Values: []*storage.RuleValue{
							{Value: "team=infra", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			hasError: true,
		},
		{
			name: "Deployment annotation rule",
			scope: &storage.EntityScope{
				Rules: []*storage.EntityScopeRule{
					{
						Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
						Field:  storage.EntityField_FIELD_ANNOTATION,
						Values: []*storage.RuleValue{
							{Value: "owner=team-a", MatchType: storage.MatchType_EXACT},
						},
					},
				},
			},
			expected:      search.NewQueryBuilder().AddMapQuery(search.DeploymentAnnotation, `"owner"`, `"team-a"`).ProtoQuery(),
			assertQueries: assertByDirectComparison,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			qb := &queryBuilder{
				entityScope: tc.scope,
			}
			result, err := qb.buildEntityScopeQuery()
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tc.assertQueries(t, tc.expected, result)
		})
	}
}
