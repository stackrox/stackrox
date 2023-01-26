//go:build sql_integration
// +build sql_integration

package postgres_test

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
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
	"github.com/stretchr/testify/suite"
)

func TestGroupByQueries(t *testing.T) {
	suite.Run(t, new(GroupByQueriesTestSuite))
}

type GroupByQueriesTestSuite struct {
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

func (s *GroupByQueriesTestSuite) SetupTest() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	pool := s.testDB.Pool

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

func (s *GroupByQueriesTestSuite) initializeTestGraph() {
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
	s.Require().NoError(s.testGrandparentStore.Upsert(testCtx, &storage.TestGrandparent{
		Id:        "3",
		Val:       "Grandparent2", // repeated value to test group by
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
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "4",
		ParentId: "1",
		Val:      "TestParent11", // repeat the values
		Children: []*storage.TestParent1_Child1Ref{},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "5",
		ParentId: "1",
		Val:      "TestParent12", // repeat the values
		Children: []*storage.TestParent1_Child1Ref{},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "6",
		ParentId: "2",
		Val:      "TestParent13", // repeat the values
		Children: []*storage.TestParent1_Child1Ref{},
	}))
	s.Require().NoError(s.testParent1Store.Upsert(testCtx, &storage.TestParent1{
		Id:       "7",
		ParentId: "2",
		Val:      "TestParent13", // repeat the values
		Children: []*storage.TestParent1_Child1Ref{},
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

func (s *GroupByQueriesTestSuite) mustRunQuery(typeName string, q *v1.Query) []search.Result {
	res, err := postgres.RunSearchRequestForSchema(testCtx, getTestSchema(s.T(), typeName), q, s.testDB.Pool)
	s.Require().NoError(err)
	return res
}

func (s *GroupByQueriesTestSuite) mustRunCountQuery(typeName string, q *v1.Query) int {
	count, err := postgres.RunCountRequestForSchema(ctx, getTestSchema(s.T(), typeName), q, s.testDB.Pool)
	s.Require().NoError(err)
	return count
}

func (s *GroupByQueriesTestSuite) assertResultsHaveIDs(results []search.Result, orderMatters bool, expectedIDs ...string) {
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

func (s *GroupByQueriesTestSuite) runTestCases(testCases []graphQueryTestCase) {
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
			if testCase.queryType == postgres.COUNT {
				s.Equal(len(testCase.expectedResultIDs), s.mustRunCountQuery(testCase.queriedProtoType, q))
			} else {
				res := s.mustRunQuery(testCase.queriedProtoType, q)
				s.assertResultsHaveIDs(res, testCase.orderMatters, testCase.expectedResultIDs...)
			}
		})
	}
}

func (s *GroupByQueriesTestSuite) TestDerivedGroupByPrimaryKey() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:              "one-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStrings(search.TestParent1Count, ">1").ProtoQuery(),
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "two-hop count",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStrings(search.TestChild1Count, ">1").ProtoQuery(),
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:              "two-hop count again",
			queriedProtoType:  "testgrandparent",
			q:                 search.NewQueryBuilder().AddStrings(search.TestChild1Count, ">5").ProtoQuery(),
			expectedResultIDs: []string{},
		},
		{
			desc:              "no-hop count; count = 1",
			queriedProtoType:  "testparent1",
			q:                 search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, "=1").ProtoQuery(),
			expectedResultIDs: []string{"1", "2", "3", "4", "5", "6", "7"},
		},
		{
			desc:              "no-hop count; count > 1",
			queriedProtoType:  "testparent1",
			q:                 search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">1").ProtoQuery(),
			expectedResultIDs: []string{},
		},
	})
}

func (s *GroupByQueriesTestSuite) TestDerivedGroupByNonPrimaryKey() {
	s.runTestCases([]graphQueryTestCase{
		{
			desc:             "one-hop count",
			queriedProtoType: "testgrandparent",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1Count, ">1").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestGrandparentVal.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:             "two-hop count",
			queriedProtoType: "testgrandparent",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestChild1Count, ">1").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestGrandparentVal.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{"1", "2"},
		},
		{
			desc:             "two-hop count again",
			queriedProtoType: "testgrandparent",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestChild1Count, ">5").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestGrandparentVal.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{},
		},
		{
			desc:             "no-hop count; count = 1",
			queriedProtoType: "testparent1",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, "=1").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestParent1Val.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{},
		},
		{
			desc:             "no-hop count; count > 1",
			queriedProtoType: "testparent1",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">1").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestParent1Val.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{"1", "2", "3", "4", "5", "6", "7"},
		},
		{
			desc:             "no-hop count; count > 2",
			queriedProtoType: "testparent1",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">2").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestParent1Val.String()},
				}
				return q
			}(),
			expectedResultIDs: []string{"3", "6", "7"},
		},
	})
}

func (s *GroupByQueriesTestSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}
