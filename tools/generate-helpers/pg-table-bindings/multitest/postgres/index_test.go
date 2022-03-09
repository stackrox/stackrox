//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = context.Background()
)

type IndexSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator

	pool    *pgxpool.Pool
	store   Store
	indexer *indexerImpl
}

func TestIndex(t *testing.T) {
	suite.Run(t, new(IndexSuite))
}

func (s *IndexSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres index tests")
		s.T().SkipNow()
	}

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = pgxpool.ConnectConfig(context.Background(), config)
	s.Require().NoError(err)

	Destroy(ctx, s.pool)
	s.store = New(ctx, s.pool)
	s.indexer = NewIndexer(s.pool)
}

func (s *IndexSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	s.pool.Close()
}

func (s *IndexSuite) getStruct(i int, f func(s *storage.TestMultiKeyStruct)) *storage.TestMultiKeyStruct {
	out := &storage.TestMultiKeyStruct{
		Key1: fmt.Sprintf("key1%d", i),
		Key2: fmt.Sprintf("key2%d", i),
	}
	f(out)
	s.Require().NoError(s.store.Upsert(ctx, out))
	return out
}

func getID(s *storage.TestMultiKeyStruct) string {
	return s.Key1 + "+" + s.Key2
}

func (s *IndexSuite) TestBool() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Bool = false
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Bool = true
	})
	_, _ = testStruct0, testStruct1
	res, err := s.indexer.Search(search.NewQueryBuilder().AddBools(search.TestBool, false).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Len(res, 1)
	s.Equal(getID(testStruct0), res[0].ID)
}

func (s *IndexSuite) TestStringSlice() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"yeah", "no"}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"whatever", "blah", "yeahyeah"}
	})

	_, _ = testStruct0, testStruct1
	for _, testCase := range []struct {
		desc            string
		q               *v1.Query
		expectedResults []*storage.TestMultiKeyStruct
	}{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, "yeah").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		/* TODO
		{
			desc:            "prefix",
			q:               search.NewQueryBuilder().AddStrings(search.TestStringSlice, "yeah").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
		{
			desc:            "prefix matches only one",
			q:               search.NewQueryBuilder().AddStrings(search.TestStringSlice, "what").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            "regex",
			q:               search.NewQueryBuilder().AddRegexes(search.TestStringSlice, "bl.*").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		*/
	} {
		s.Run(testCase.desc, func() {
			results, err := s.indexer.Search(testCase.q)
			s.Require().NoError(err)

			actualIDs := make([]string, 0, len(results))
			for _, res := range results {
				actualIDs = append(actualIDs, res.ID)
			}

			expectedIDs := make([]string, 0, len(testCase.expectedResults))
			for _, s := range testCase.expectedResults {
				expectedIDs = append(expectedIDs, getID(s))
			}
			s.ElementsMatch(actualIDs, expectedIDs)
		})
	}
}
