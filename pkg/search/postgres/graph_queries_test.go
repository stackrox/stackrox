package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"

	testChild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testchild1"
	testChild2 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testchild2"
	testG2Grandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testg2grandchild1"
	testG3Grandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testg3grandchild1"
	testGGrandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testggrandchild1"
	testGrandchild1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testgrandchild1"
	testGrandparent "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testgrandparent"
	testParent1 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent1"
	testParent2 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent2"
	testParent3 "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testgraphtables/testparent3"
)

var (
	testCtx = sac.WithAllAccess(context.Background())
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
	testGrandChild1Store   testGrandchild1.Store
	testGGranchild1Store   testGGrandchild1.Store
	testG2Grandchild1Store testG2Grandchild1.Store
	testG3Grandchild1Store testG3Grandchild1.Store
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

	s.pool = pool
	s.testGrandparentStore = testGrandparent.New(testCtx, pool)
	s.testGrandparentStore = testGrandparent.New(testCtx, pool)
	s.testChild1Store = testChild1.New(testCtx, pool)
	s.testChild2Store = testChild2.New(testCtx, pool)
	s.testParent1Store = testParent1.New(testCtx, pool)
	s.testParent2Store = testParent2.New(testCtx, pool)
	s.testParent3Store = testParent3.New(testCtx, pool)
	s.testGrandChild1Store = testGrandchild1.New(testCtx, pool)
	s.testGGranchild1Store = testGGrandchild1.New(testCtx, pool)
	s.testG2Grandchild1Store = testG2Grandchild1.New(testCtx, pool)
	s.testG3Grandchild1Store = testG3Grandchild1.New(testCtx, pool)
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
		},
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
}

func (s *GraphQueriesTestSuite) TestFirst() {
}

func (s *GraphQueriesTestSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}
