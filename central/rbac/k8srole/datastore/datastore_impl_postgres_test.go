//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/rbac/k8srole/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestK8SRoleDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(K8SRolePostgresDataStoreTestSuite))
}

type K8SRolePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        postgres.DB
	datastore DataStore
}

func (s *K8SRolePostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *K8SRolePostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	roleStorage := pgStore.New(s.db)
	s.datastore = New(roleStorage)
}

func (s *K8SRolePostgresDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *K8SRolePostgresDataStoreTestSuite) TestSearchRoles() {
	ctx := sac.WithAllAccess(context.Background())

	cluster1Id := testconsts.Cluster1
	cluster2Id := testconsts.Cluster2
	role1 := &storage.K8SRole{
		Id:          uuid.NewV4().String(),
		Name:        "cluster-admin",
		Namespace:   "default",
		ClusterId:   cluster1Id,
		ClusterName: "test-cluster-1",
	}

	role2 := &storage.K8SRole{
		Id:          uuid.NewV4().String(),
		Name:        "read-only",
		Namespace:   "kube-system",
		ClusterId:   cluster1Id,
		ClusterName: "test-cluster-1",
	}

	role3 := &storage.K8SRole{
		Id:          uuid.NewV4().String(),
		Name:        "system:node",
		Namespace:   "default",
		ClusterId:   cluster2Id,
		ClusterName: "test-cluster-2",
	}

	// Add roles
	err := s.datastore.UpsertRole(ctx, role1)
	s.NoError(err)
	err = s.datastore.UpsertRole(ctx, role2)
	s.NoError(err)
	err = s.datastore.UpsertRole(ctx, role3)
	s.NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
		expectedNames []string
	}{
		{
			name:          "empty query returns all roles with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{role1.GetId(), role2.GetId(), role3.GetId()},
			expectedNames: []string{"cluster-admin", "read-only", "system:node"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedNames: []string{"cluster-admin", "read-only", "system:node"},
		},
		{
			name:          "query by cluster ID",
			query:         pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster1Id).ProtoQuery(),
			expectedCount: 2,
			expectedIDs:   []string{role1.GetId(), role2.GetId()},
			expectedNames: []string{"cluster-admin", "read-only"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchRoles(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				actualNames = append(actualNames, result.GetName())
				s.Equal(v1.SearchCategory_ROLES, result.GetCategory())
			}

			if len(tc.expectedIDs) > 0 {
				s.ElementsMatch(tc.expectedIDs, actualIDs)
			}

			if len(tc.expectedNames) > 0 {
				s.ElementsMatch(tc.expectedNames, actualNames)
			}
		})
	}

	// Clean up
	s.NoError(s.datastore.RemoveRole(ctx, role1.GetId()))
	s.NoError(s.datastore.RemoveRole(ctx, role2.GetId()))
	s.NoError(s.datastore.RemoveRole(ctx, role3.GetId()))

	// Verify cleanup
	results, err := s.datastore.SearchRoles(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Empty(results)
}
