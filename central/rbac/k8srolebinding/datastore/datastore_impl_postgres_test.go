//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store/postgres"
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

func TestK8SRoleBindingDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(K8SRoleBindingPostgresDataStoreTestSuite))
}

type K8SRoleBindingPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        postgres.DB
	datastore DataStore
}

func (s *K8SRoleBindingPostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *K8SRoleBindingPostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	roleBindingStorage := pgStore.New(s.db)
	s.datastore = New(roleBindingStorage)
}

func (s *K8SRoleBindingPostgresDataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *K8SRoleBindingPostgresDataStoreTestSuite) TestSearchRoleBindings() {
	ctx := sac.WithAllAccess(context.Background())

	// Generate UUIDs for test data
	cluster1ID := testconsts.Cluster1
	cluster2ID := testconsts.Cluster2

	// Create test role bindings
	binding1 := &storage.K8SRoleBinding{
		Id:          uuid.NewV4().String(),
		Name:        "admin-binding",
		Namespace:   "default",
		ClusterId:   cluster1ID,
		ClusterName: "test-cluster-1",
		RoleId:      uuid.NewV4().String(),
	}

	binding2 := &storage.K8SRoleBinding{
		Id:          uuid.NewV4().String(),
		Name:        "read-binding",
		Namespace:   "kube-system",
		ClusterId:   cluster1ID,
		ClusterName: "test-cluster-1",
		RoleId:      uuid.NewV4().String(),
	}

	binding3 := &storage.K8SRoleBinding{
		Id:          uuid.NewV4().String(),
		Name:        "node-binding",
		Namespace:   "default",
		ClusterId:   cluster2ID,
		ClusterName: "test-cluster-2",
		RoleId:      uuid.NewV4().String(),
	}

	// Add role bindings
	err := s.datastore.UpsertRoleBinding(ctx, binding1)
	s.NoError(err)
	err = s.datastore.UpsertRoleBinding(ctx, binding2)
	s.NoError(err)
	err = s.datastore.UpsertRoleBinding(ctx, binding3)
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
			name:          "empty query returns all role bindings with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{binding1.GetId(), binding2.GetId(), binding3.GetId()},
			expectedNames: []string{"admin-binding", "read-binding", "node-binding"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedNames: []string{"admin-binding", "read-binding", "node-binding"},
		},
		{
			name:          "query by cluster ID",
			query:         pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster1ID).ProtoQuery(),
			expectedCount: 2,
			expectedIDs:   []string{binding1.GetId(), binding2.GetId()},
			expectedNames: []string{"admin-binding", "read-binding"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchRoleBindings(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				actualNames = append(actualNames, result.GetName())
				s.Equal(v1.SearchCategory_ROLEBINDINGS, result.GetCategory())
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
	s.NoError(s.datastore.RemoveRoleBinding(ctx, binding1.GetId()))
	s.NoError(s.datastore.RemoveRoleBinding(ctx, binding2.GetId()))
	s.NoError(s.datastore.RemoveRoleBinding(ctx, binding3.GetId()))

	// Verify cleanup
	results, err := s.datastore.SearchRoleBindings(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Empty(results)
}
