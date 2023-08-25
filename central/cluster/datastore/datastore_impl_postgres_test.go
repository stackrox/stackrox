//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	namespace "github.com/stackrox/rox/central/namespace/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	fakeClusterID   = "FAKECLUSTERID"
	mainImage       = "docker.io/stackrox/rox:latest"
	centralEndpoint = "central.stackrox:443"
)

func TestClusterDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ClusterPostgresDataStoreTestSuite))
}

type ClusterPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx              context.Context
	db               *pgtest.TestPostgres
	nsDatastore      namespace.DataStore
	clusterDatastore DataStore
}

func (s *ClusterPostgresDataStoreTestSuite) SetupSuite() {

	s.ctx = sac.WithAllAccess(context.Background())

	s.db = pgtest.ForT(s.T())

	ds, err := namespace.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.nsDatastore = ds
	clusterDS, err := GetTestPostgresDataStore(s.T(), s.db.DB)

	s.NoError(err)
	s.clusterDatastore = clusterDS
}

func (s *ClusterPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchClusterStatus() {
	ctx := sac.WithAllAccess(context.Background())

	// At some point in the postgres migration, the following query did trigger an error
	// because of a missing options map in the cluster health status schema.
	// This test is there to ensure the search does not end in error for technical reasons.
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterStatus, storage.ClusterHealthStatus_UNHEALTHY.String()).ProtoQuery()
	res, err := s.clusterDatastore.Search(ctx, query)
	s.NoError(err)
	s.Equal(0, len(res))
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())

	// Upsert cluster.
	c1ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c1",
		Labels:             map[string]string{"env": "prod", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	// Upsert cluster.
	c2ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c2",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	ns1C1 := fixtures.GetNamespace(c1ID, "c1", "n1")
	ns2C1 := fixtures.GetNamespace(c1ID, "c1", "n2")
	ns1C2 := fixtures.GetNamespace(c2ID, "c2", "n1")

	// Upsert namespaces.
	s.NoError(s.nsDatastore.AddNamespace(ctx, ns1C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns2C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns1C2))

	for _, tc := range []struct {
		desc        string
		ctx         context.Context
		query       *v1.Query
		expectedIDs []string
		queryNs     bool
	}{
		{
			desc:  "Search clusters with empty query",
			ctx:   ctx,
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID, c2ID},
		},
		{
			desc:  "Search clusters with cluster query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, c1ID).ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:        "Search clusters with namespace query",
			ctx:         ctx,
			query:       pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster+namespace query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),

			expectedIDs: []string{c1ID, c2ID},
		},
		{
			desc:  "Search clusters with cluster scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
		},
		{
			desc:  "Search clusters with cluster scope and in-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and out-of-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c2ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),

			expectedIDs: []string{},
		},
		{
			desc:  "Search clusters with namespace scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with namespace scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with namespace scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
		},

		{
			desc:  "Search namespaces with empty query",
			ctx:   ctx,
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id, ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:        "Search namespaces with cluster query",
			ctx:         ctx,
			query:       pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			expectedIDs: []string{ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster+namespace query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id, ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster+namespace non-matching search fields",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "blah").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and in-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and out-of-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc: "Search namespaces with cluster+namespace scope",
			ctx: scoped.Context(ctx,
				scoped.Scope{
					ID:    ns1C1.Id,
					Level: v1.SearchCategory_NAMESPACES,
					Parent: &scoped.Scope{
						ID:    c1ID,
						Level: v1.SearchCategory_CLUSTERS,
					},
				},
			),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc: "Search namespaces with cluster+namespace scope and out-of-scope cluster query",
			ctx: scoped.Context(ctx,
				scoped.Scope{
					ID:    ns1C1.Id,
					Level: v1.SearchCategory_NAMESPACES,
					Parent: &scoped.Scope{
						ID:    c1ID,
						Level: v1.SearchCategory_CLUSTERS,
					},
				},
			),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			var actual []pkgSearch.Result
			var err error
			if tc.queryNs {
				actual, err = s.nsDatastore.Search(tc.ctx, tc.query)
			} else {
				actual, err = s.clusterDatastore.Search(tc.ctx, tc.query)
			}
			assert.NoError(t, err)
			assert.Len(t, actual, len(tc.expectedIDs))
			actualIDs := pkgSearch.ResultsToIDs(actual)
			assert.ElementsMatch(t, tc.expectedIDs, actualIDs)
		})
	}
}
