//go:build sql_integration
// +build sql_integration

package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	testChild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testchild1"
	testChild1P4 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testchild1p4"
	testChild2 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testchild2"
	testG2Grandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testg2grandchild1"
	testG3Grandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testg3grandchild1"
	testGGrandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testggrandchild1"
	testGrandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testgrandchild1"
	testGrandparent "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testgrandparent"
	testParent1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent1"
	testParent2 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent2"
	testParent3 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent3"
	testParent4 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent4"
	testShortCircuit "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testshortcircuit"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testCtx = sac.WithAllAccess(context.Background())

	getTestSchema = func() func(t *testing.T, typeName string) *walker.Schema {
		relevantSchemas := make(map[string]*walker.Schema)
		allSchemas := mapping.GetAllRegisteredSchemas()
		for _, schema := range allSchemas {
			lowerTypeName := strings.ToLower(schema.TypeName)
			if strings.HasPrefix(lowerTypeName, "test") {
				relevantSchemas[lowerTypeName] = schema
			}
		}
		return func(t *testing.T, typeName string) *walker.Schema {
			schema := relevantSchemas[typeName]
			require.NotNil(t, schema, "No schema registered for %s (registered schemas: %+v)", typeName, relevantSchemas)
			return schema
		}
	}()
)

func TestGraphQueries(t *testing.T) {
	suite.Run(t, new(GraphQueriesTestSuite))
}

type GraphQueriesTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	pool        *pgxpool.Pool

	testGrandparentStore   testGrandparent.Store
	testChild1Store        testChild1.Store
	testChild2Store        testChild2.Store
	testParent1Store       testParent1.Store
	testParent2Store       testParent2.Store
	testParent3Store       testParent3.Store
	testParent4Store       testParent4.Store
	testChild1P4Store      testChild1P4.Store
	testGrandChild1Store   testGrandchild1.Store
	testGGrandchild1Store  testGGrandchild1.Store
	testG2Grandchild1Store testG2Grandchild1.Store
	testG3Grandchild1Store testG3Grandchild1.Store
	testShortCircuitStore  testShortCircuit.Store
}

func (s *GraphQueriesTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(testCtx, config)
	s.Require().NoError(err)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	s.pool = pool
	s.testGrandparentStore = testGrandparent.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testChild1Store = testChild1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testChild2Store = testChild2.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testParent1Store = testParent1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testParent2Store = testParent2.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testParent3Store = testParent3.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testParent4Store = testParent4.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testChild1P4Store = testChild1P4.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testGrandChild1Store = testGrandchild1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testGGrandchild1Store = testGGrandchild1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testG2Grandchild1Store = testG2Grandchild1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testG3Grandchild1Store = testG3Grandchild1.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.testShortCircuitStore = testShortCircuit.CreateTableAndNewStore(testCtx, pool, gormDB)
	s.initializeTestGraph()
}

func (s *GraphQueriesTestSuite) initializeTestGraph() {
	s.Require().NoError(s.testGrandparentStore.Upsert(testCtx, &storage.TestGrandparent{
		Id:  "1",
		Val: "Grandparent1",
		Embedded: []*storage.TestGrandparent_Embedded{
			{Val: "Grandparent1Embedded1", Embedded2: []*storage.TestGrandparent_Embedded_Embedded2{
				{Val: "Grandparent1Embedded11"},
				{Val: "Grandparent1Embedded12"},
			}},
			{Val: "Grandparent1Embedded2"},
		},
	}))
	s.Require().NoError(s.testGrandparentStore.Upsert(testCtx, &storage.TestGrandparent{
		Id:  "2",
		Val: "Grandparent2",
		Embedded: []*storage.TestGrandparent_Embedded{
			{Val: "Grandparent2Embedded1", Embedded2: []*storage.TestGrandparent_Embedded_Embedded2{
				{Val: "Grandparent2Embedded11"},
			}},
			{Val: "Grandparent2Embedded2", Embedded2: []*storage.TestGrandparent_Embedded_Embedded2{
				{Val: "Grandparent2Embedded21"},
			}},
		},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "1",
		ParentId: "1",
		Val:      "TestParent11",
		Children: []*storage.TestParent1_Child1Ref{
			{ChildId: "1"},
			{ChildId: "2"},
		},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "2",
		ParentId: "1",
		Val:      "TestParent12",
		Children: []*storage.TestParent1_Child1Ref{
			{ChildId: "1"},
			{ChildId: "3"},
		},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "3",
		ParentId: "2",
		Val:      "TestParent13",
		Children: []*storage.TestParent1_Child1Ref{
			{ChildId: "3"},
			{ChildId: "4"},
			{ChildId: "5"},
		},
	}))
	s.Require().NoError(s.testParent4Store.Upsert(testCtx, &storage.TestParent4{
		Id:       "4",
		ParentId: "1",
		Val:      "TestParent4",
	}))
	s.Require().NoError(s.testChild1Store.Upsert(testCtx, &storage.TestChild1{
		Id:  "1",
		Val: "Child11",
	}))
	s.Require().NoError(s.testChild1Store.Upsert(testCtx, &storage.TestChild1{
		Id:  "2",
		Val: "Child12",
	}))
	s.Require().NoError(s.testChild1Store.Upsert(testCtx, &storage.TestChild1{
		Id:  "3",
		Val: "Child13",
	}))
	s.Require().NoError(s.testChild1Store.Upsert(testCtx, &storage.TestChild1{
		Id:  "4",
		Val: "Child14",
	}))
	s.Require().NoError(s.testChild1Store.Upsert(testCtx, &storage.TestChild1{
		Id:  "5",
		Val: "Child15",
	}))
	s.Require().NoError(s.testChild1P4Store.Upsert(testCtx, &storage.TestChild1P4{
		Id:       "C1P4",
		ParentId: "4",
		Val:      "Child1P4",
	}))

	s.Require().NoError(s.testGrandChild1Store.Upsert(testCtx, &storage.TestGrandChild1{
		Id:       "1",
		ParentId: "1",
		ChildId:  "1",
		Val:      "Grandchild11",
	}))
	s.Require().NoError(s.testGrandChild1Store.Upsert(testCtx, &storage.TestGrandChild1{
		Id:       "2",
		ParentId: "1",
		ChildId:  "2",
		Val:      "Grandchild12",
	}))
	s.Require().NoError(s.testGrandChild1Store.Upsert(testCtx, &storage.TestGrandChild1{
		Id:       "3",
		ParentId: "2",
		ChildId:  "3",
		Val:      "Grandchild13",
	}))
	s.Require().NoError(s.testGGrandchild1Store.Upsert(testCtx, &storage.TestGGrandChild1{
		Id:  "1",
		Val: "GGrandchild11",
	}))
	s.Require().NoError(s.testGGrandchild1Store.Upsert(testCtx, &storage.TestGGrandChild1{
		Id:  "2",
		Val: "GGrandchild11",
	}))
	s.Require().NoError(s.testGGrandchild1Store.Upsert(testCtx, &storage.TestGGrandChild1{
		Id:  "3",
		Val: "GGrandchild11",
	}))
	s.Require().NoError(s.testG2Grandchild1Store.Upsert(testCtx, &storage.TestG2GrandChild1{
		Id:       "5",
		ParentId: "3",
		ChildId:  "10",
		Val:      "g2GrandChild15",
	}))
	s.Require().NoError(s.testG2Grandchild1Store.Upsert(testCtx, &storage.TestG2GrandChild1{
		Id:       "6",
		ParentId: "3",
		ChildId:  "10",
		Val:      "g2GrandChild16",
	}))
	s.Require().NoError(s.testShortCircuitStore.Upsert(testCtx, &storage.TestShortCircuit{
		Id:             "3",
		ChildId:        "5",
		G2GrandchildId: "5",
	}))
}

func (s *GraphQueriesTestSuite) mustRunQuery(typeName string, q *v1.Query) []search.Result {
	res, err := postgres.RunSearchRequestForSchema(getTestSchema(s.T(), typeName), q, s.pool)
	s.Require().NoError(err)
	return res
}

func (s *GraphQueriesTestSuite) assertResultsHaveIDs(results []search.Result, orderMatters bool, expectedIDs ...string) {
	idsFromResult := make([]string, 0, len(results))
	for _, res := range results {
		idsFromResult = append(idsFromResult, res.ID)
	}
	if orderMatters {
		s.Equal(idsFromResult, expectedIDs)
	} else {
		s.ElementsMatch(idsFromResult, expectedIDs)
	}
}

type graphQueryTestCase struct {
	desc        string
	queriedType string

	// Passing queryStrings is short for passing
	// search.NewQueryBuilder().AddStrings() with the values
	// in queryStrings.
	// Exactly one of q and queryStrings must be provided.
	q            *v1.Query
	queryStrings map[search.FieldLabel][]string

	expectedResultIDs []string

	orderMatters bool
}

func (s *GraphQueriesTestSuite) runTestCases(testCases []graphQueryTestCase) {
	for _, testCase := range testCases {
		s.Run(testCase.desc, func() {
			q := testCase.q
			if q == nil {
				s.Require().NotEmpty(testCase.queryStrings, "neither query nor queryStrings specified")
				qb := search.NewQueryBuilder()
				for k, v := range testCase.queryStrings {
					qb.AddStrings(k, v...)
				}
				q = qb.ProtoQuery()
			} else {
				s.Require().Empty(testCase.queryStrings, "both query and queryStrings specified")
			}
			res := s.mustRunQuery(testCase.queriedType, q)
			s.assertResultsHaveIDs(res, testCase.orderMatters, testCase.expectedResultIDs...)
		})
	}
}

func (s *GraphQueriesTestSuite) TestQueriesOnGrandParentValue() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "simple grandparent query",
			queriedType:       "testgrandparent",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1"},
		},
		{
			desc:              "query from parent",
			queriedType:       "testparent1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "query from child",
			queriedType:       "testchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2", "3"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestShortCircuit() {
	s.runTestCases([]graphQueryTestCase{
		// Test short circuit as path
		{
			desc:              "no query",
			queriedType:       "testshortcircuit",
			q:                 search.NewQueryBuilder().ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		{
			desc:              "one query - one table - 1 match",
			queriedType:       "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "Child15").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		{
			desc:              "one query - one table - match",
			queriedType:       "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "no match").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{},
		},
		{
			desc:              "two queries - two tables",
			queriedType:       "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "Child15").AddExactMatches(search.TestG2Grandchild1Val, "g2GrandChild15").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		// Test short circuit as fastest path (but does not exist)
		{
			desc:              "query that _would_ pass through short circuit",
			queriedType:       "testchild1",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestG2Grandchild1Val, "g2GrandChild16").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"2"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestDerived() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "one-hop count",
			queriedType:       "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestParent1Count.String()}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"2", "1"},
		},
		{
			desc:              "one-hop count (reversed)",
			queriedType:       "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestParent1Count.String(), Reversed: true}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "two-hop count",
			queriedType:       "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestChild1Count.String()}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"2", "1"},
		},
		{
			desc:              "two-hop count (reversed)",
			queriedType:       "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestChild1Count.String(), Reversed: true}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"1", "2"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestSubGraphSearch() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "query out-of-scope resource from parent4",
			queriedType:       "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent2ID: {"r/.*1"}},
			expectedResultIDs: []string{},
		},
		{
			desc:              "query out-of-scope resource from child1p4",
			queriedType:       "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1ID: {"r/.*1"}},
			expectedResultIDs: []string{},
		},
		{
			desc:              "query in-scope resource from parent4",
			queriedType:       "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4Val: {"r/.*4"}},
			expectedResultIDs: []string{"4"},
		},
		{
			desc:              "query in-scope child from parent4",
			queriedType:       "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1P4ID: {"r/.*P4"}},
			expectedResultIDs: []string{"4"},
		},
		{
			desc:              "query out-of-scope parent from child1p4",
			queriedType:       "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4ID: {"r/.*4"}},
			expectedResultIDs: []string{},
		},
	})
}

func (s *GraphQueriesTestSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}
