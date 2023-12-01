//go:build sql_integration

package postgres_test

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
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

const (
	parent4ID = "44444444-4444-4011-0000-444444444444"
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

	testDB *pgtest.TestPostgres

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

	s.testDB = pgtest.ForT(s.T())
	pool := s.testDB.DB

	s.testGrandparentStore = testGrandparent.New(pool)
	s.testChild1Store = testChild1.New(pool)
	s.testChild2Store = testChild2.New(pool)
	s.testParent1Store = testParent1.New(pool)
	s.testParent2Store = testParent2.New(pool)
	s.testParent3Store = testParent3.New(pool)
	s.testParent4Store = testParent4.New(pool)
	s.testChild1P4Store = testChild1P4.New(pool)
	s.testGrandChild1Store = testGrandchild1.New(pool)
	s.testGGrandchild1Store = testGGrandchild1.New(pool)
	s.testG2Grandchild1Store = testG2Grandchild1.New(pool)
	s.testG3Grandchild1Store = testG3Grandchild1.New(pool)
	s.testShortCircuitStore = testShortCircuit.New(pool)
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
				{Val: "GrandparentEmbeddedShared"},
			}},
			{Val: "Grandparent1Embedded2"},
		},
		RiskScore: 10,
	}))
	s.Require().NoError(s.testGrandparentStore.Upsert(testCtx, &storage.TestGrandparent{
		Id:  "2",
		Val: "Grandparent2",
		Embedded: []*storage.TestGrandparent_Embedded{
			{Val: "Grandparent2Embedded1", Embedded2: []*storage.TestGrandparent_Embedded_Embedded2{
				{Val: "Grandparent2Embedded11"},
				{Val: "GrandparentEmbeddedShared"},
			}},
			{Val: "Grandparent2Embedded2", Embedded2: []*storage.TestGrandparent_Embedded_Embedded2{
				{Val: "Grandparent2Embedded21"},
				{Val: "GrandparentEmbeddedShared"},
			}},
		},
		RiskScore: 20,
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "1",
		ParentId: "1",
		Val:      "TestParent11",
		Children: []*storage.TestParent1_Child1Ref{
			{ChildId: "1"},
			{ChildId: "2"},
		},
		StringSlice: []string{"a", "b", "c"},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "2",
		ParentId: "1",
		Val:      "TestParent12",
		Children: []*storage.TestParent1_Child1Ref{
			{ChildId: "1"},
			{ChildId: "3"},
		},
		StringSlice: []string{"a", "b", "c"},
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
		StringSlice: []string{"a", "b", "c", "d", "e"},
	}))
	s.Require().NoError(s.testParent4Store.Upsert(testCtx, &storage.TestParent4{
		Id:       parent4ID,
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
		ParentId: parent4ID,
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

func (s *GraphQueriesTestSuite) mustRunCountQuery(typeName string, q *v1.Query) int {
	count, err := pgSearch.RunCountRequestForSchema(ctx, getTestSchema(s.T(), typeName), q, s.testDB.DB)
	s.Require().NoError(err)
	return count
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
	desc             string
	queriedProtoType string
	queryType        pgSearch.QueryType
	// Passing queryStrings is short for passing
	// search.NewQueryBuilder().AddStrings() with the values
	// in queryStrings.
	// Exactly one of q and queryStrings must be provided.
	q            *v1.Query
	queryStrings map[search.FieldLabel][]string

	expectedResultIDs []string
	expectedError     bool
	orderMatters      bool
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
			if testCase.queryType == pgSearch.COUNT {
				s.Equal(len(testCase.expectedResultIDs), s.mustRunCountQuery(testCase.queriedProtoType, q))
			} else {
				res, err := pgSearch.RunSearchRequestForSchema(testCtx, getTestSchema(s.T(), testCase.queriedProtoType), q, s.testDB.DB)
				if testCase.expectedError {
					s.Error(err)
					return
				}
				s.NoError(err)
				s.assertResultsHaveIDs(res, testCase.orderMatters, testCase.expectedResultIDs...)
			}
		})
	}
}

func (s *GraphQueriesTestSuite) TestQueriesOnGrandParentValue() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "simple grandparent query",
			queriedProtoType:  "testgrandparent",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1"},
		},
		{
			desc:              "query from parent",
			queriedProtoType:  "testparent1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "query from child",
			queriedProtoType:  "testchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2", "3"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestCountQueriesOnGrandParentValue() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "simple grandparent query",
			queriedProtoType:  "testgrandparent",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query from parent",
			queriedProtoType:  "testparent1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query from child",
			queriedProtoType:  "testchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2", "3"},
			queryType:         pgSearch.COUNT,
		},
	})
}

func (s *GraphQueriesTestSuite) TestCountQueriesOnGrandChild() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "simple grandchild query",
			queriedProtoType:  "testgrandchild1",
			q:                 search.EmptyQuery(),
			expectedResultIDs: []string{"1", "2", "3"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "grand parent query",
			queriedProtoType:  "testgrandchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentVal: {"r/.*1"}},
			expectedResultIDs: []string{"1", "2", "3"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:             "grand child query + grand parent query",
			queriedProtoType: "testgrandchild1",
			queryStrings: map[search.FieldLabel][]string{
				search.TestGrandparentVal: {"r/.*1"},
				search.TestGrandchild1ID:  {"1"},
			},
			expectedResultIDs: []string{"1"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:             "non-overlapping grand child query and grand parent query",
			queriedProtoType: "testgrandchild1",
			queryStrings: map[search.FieldLabel][]string{
				search.TestGrandparentVal: {"r/.*2"},
				search.TestGrandchild1ID:  {"1"},
			},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "embedded grand parent query",
			queriedProtoType:  "testgrandchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentEmbedded2: {"Grandparent1Embedded11"}},
			expectedResultIDs: []string{"1", "2", "3"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "non-overlapping embedded grand parent query",
			queriedProtoType:  "testgrandchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentEmbedded2: {"Grandparent2Embedded11"}},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "shared value embedded grand parent query",
			queriedProtoType:  "testgrandchild1",
			queryStrings:      map[search.FieldLabel][]string{search.TestGrandparentEmbedded2: {"GrandparentEmbeddedShared"}},
			expectedResultIDs: []string{"1", "2", "3"},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:             "scoped shared value embedded grand parent query",
			queriedProtoType: "testgrandchild1",
			queryStrings: map[search.FieldLabel][]string{
				search.TestGrandparentVal:       {"r/.*2"},
				search.TestGrandparentEmbedded2: {"GrandparentEmbeddedShared"},
			},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
	})
}

func (s *GraphQueriesTestSuite) TestShortCircuit() {
	s.runTestCases([]graphQueryTestCase{
		// Test short circuit as path
		{
			desc:              "no query",
			queriedProtoType:  "testshortcircuit",
			q:                 search.NewQueryBuilder().ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		{
			desc:              "one query - one table - 1 match",
			queriedProtoType:  "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "Child15").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		{
			desc:              "one query - one table - match",
			queriedProtoType:  "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "no match").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{},
		},
		{
			desc:              "two queries - two tables",
			queriedProtoType:  "testshortcircuit",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestChild1Val, "Child15").AddExactMatches(search.TestG2Grandchild1Val, "g2GrandChild15").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"3"},
		},
		// Test short circuit as fastest path (but does not exist)
		{
			desc:              "query that _would_ pass through short circuit",
			queriedProtoType:  "testchild1",
			q:                 search.NewQueryBuilder().AddExactMatches(search.TestG2Grandchild1Val, "g2GrandChild16").ProtoQuery(),
			orderMatters:      false,
			expectedResultIDs: []string{"2"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestDerivedPagination() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "one-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestParent1Count.String()}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"2", "1"},
		},
		// This is unit test that demonstrates that count on array data types does not function as expected.
		// The expectation is count of individual values instead of count of arrays.
		// Remove the `validateDerivedFieldDataType` check and run this test.
		// {
		//	desc:              "one-hop count",
		//	queriedProtoType:  "testgrandparent",
		//	q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestParent1StringSliceCount.String()}}}},
		//	orderMatters:      true,
		//	expectedResultIDs: []string{"2", "1"},
		// },
		{
			desc:              "one-hop count (reversed)",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestParent1Count.String(), Reversed: true}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "two-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestChild1Count.String()}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"2", "1"},
		},
		{
			desc:              "two-hop count (reversed)",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestChild1Count.String(), Reversed: true}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "priority sorting",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestGrandParentPriority.String()}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"2", "1"},
		},
		{
			desc:              "priority sorting reversed",
			queriedProtoType:  "testgrandparent",
			q:                 &v1.Query{Pagination: &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{Field: search.TestGrandParentPriority.String(), Reversed: true}}}},
			orderMatters:      true,
			expectedResultIDs: []string{"1", "2"},
		},
	})
}

func (s *GraphQueriesTestSuite) TestSubGraphSearch() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "query out-of-scope resource from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent2ID: {"r/.*1"}},
			expectedResultIDs: []string{},
		},
		{
			desc:              "query out-of-scope resource from child1p4",
			queriedProtoType:  "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1ID: {"r/.*1"}},
			expectedResultIDs: []string{},
		},
		{
			desc:              "query in-scope resource from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4Val: {"r/.*4"}},
			expectedResultIDs: []string{parent4ID},
		},
		{
			desc:              "query in-scope child from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1P4ID: {"r/.*P4"}},
			expectedResultIDs: []string{parent4ID},
		},
		{
			desc:              "query out-of-scope parent from child1p4",
			queriedProtoType:  "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4ID: {"r/.*4"}},
			expectedResultIDs: []string{},
		},
	})
}

func (s *GraphQueriesTestSuite) TestSubGraphCountQueries() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "query out-of-scope resource from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent2ID: {"r/.*1"}},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query out-of-scope resource from child1p4",
			queriedProtoType:  "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1ID: {"r/.*1"}},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query in-scope resource from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4Val: {"r/.*4"}},
			expectedResultIDs: []string{parent4ID},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query in-scope child from parent4",
			queriedProtoType:  "testparent4",
			queryStrings:      map[search.FieldLabel][]string{search.TestChild1P4ID: {"r/.*P4"}},
			expectedResultIDs: []string{parent4ID},
			queryType:         pgSearch.COUNT,
		},
		{
			desc:              "query out-of-scope parent from child1p4",
			queriedProtoType:  "testchild1p4",
			queryStrings:      map[search.FieldLabel][]string{search.TestParent4ID: {"r/.*4"}},
			expectedResultIDs: []string{},
			queryType:         pgSearch.COUNT,
		},
	})
}

func (s *GraphQueriesTestSuite) TestDerived() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:             "one-hop count",
			queriedProtoType: "testgrandparent",
			queryStrings: map[search.FieldLabel][]string{
				search.TestParent1Count: {">1"},
			},
			expectedResultIDs: []string{"1"},
		},
		{
			desc:             "two-hop count",
			queriedProtoType: "testgrandparent",
			queryStrings: map[search.FieldLabel][]string{
				search.TestChild1Count: {">1"},
			},
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:             "two-hop count again",
			queriedProtoType: "testgrandparent",
			queryStrings: map[search.FieldLabel][]string{
				search.TestChild1Count: {">5"},
			},
			expectedResultIDs: []string{},
		},
	})
}

func (s *GraphQueriesTestSuite) TestDerivedFieldHighlighted() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "one-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStringsHighlighted(search.TestParent1Count, ">1").ProtoQuery(),
			expectedResultIDs: []string{"1"},
		},
		{
			desc:              "two-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStringsHighlighted(search.TestChild1Count, ">1").ProtoQuery(),
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "two-hop count again",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStringsHighlighted(search.TestChild1Count, ">5").ProtoQuery(),
			expectedResultIDs: []string{},
		},
		{
			desc:              "wildcard",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStringsHighlighted(search.TestChild1Count, "*").ProtoQuery(),
			expectedResultIDs: []string{"1", "2"},
			expectedError:     true,
		},
	})
}

func (s *GraphQueriesTestSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}
