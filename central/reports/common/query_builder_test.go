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
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var clusters = []effectiveaccessscope.ClusterForSAC{
	&clusterForSAC{
		ID:   uuid.NewV4().String(),
		Name: "remote",
	},
	&clusterForSAC{
		ID:   uuid.NewV4().String(),
		Name: "secured",
	},
}

var namespaces = []effectiveaccessscope.NamespaceForSAC{
	remoteNS,
	securedNS,
}

var remoteNS = &namespaceForSAC{
	ID:          "namespace1",
	Name:        "ns1",
	ClusterID:   clusters[0].GetID(),
	ClusterName: "remote",
}

var securedNS = &namespaceForSAC{
	ID:          "namespace2",
	Name:        "ns2",
	ClusterID:   clusters[1].GetID(),
	ClusterName: "secured",
}

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
							{ClusterName: clusters[1].GetName(), NamespaceName: securedNS.Name},
						},
					},
				}
				mockRole := permissionsMocks.NewMockResolvedRole(mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)
				return mockID
			},
			expectedQ: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusters[0].GetID()).ProtoQuery(),
				search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusters[1].GetID()).ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, securedNS.Name).ProtoQuery(),
				),
			),
			assertQueries: func(t testing.TB, expected *v1.Query, actual *v1.Query) {
				switch typedQ := actual.GetQuery().(type) {
				case *v1.Query_Disjunction:
					assert.ElementsMatch(t,
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
	assert.Equal(t, expected, actual)
}

// region SAC helpers

type clusterForSAC struct {
	ID     string
	Name   string
	Labels map[string]string
}

func (c *clusterForSAC) GetID() string {
	if c == nil {
		return ""
	}
	return c.ID
}

func (c *clusterForSAC) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

func (c *clusterForSAC) GetLabels() map[string]string {
	if c == nil {
		return nil
	}
	return c.Labels
}

type namespaceForSAC struct {
	ID          string
	Name        string
	ClusterID   string
	ClusterName string
	Labels      map[string]string
}

func (n *namespaceForSAC) GetID() string {
	if n == nil {
		return ""
	}
	return n.ID
}

func (n *namespaceForSAC) GetName() string {
	if n == nil {
		return ""
	}
	return n.Name
}

func (n *namespaceForSAC) GetClusterName() string {
	if n == nil {
		return ""
	}
	return n.ClusterName
}

func (n *namespaceForSAC) GetLabels() map[string]string {
	if n == nil {
		return nil
	}
	return n.Labels
}

// endregion SAC helpers
