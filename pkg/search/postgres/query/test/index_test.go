//go:build sql_integration
// +build sql_integration

package test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/test/postgres"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type SingleIndexSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator

	pool    *pgxpool.Pool
	store   postgres.Store
	indexer interface {
		Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
	}
}

func TestSingleIndex(t *testing.T) {
	suite.Run(t, new(SingleIndexSuite))
}

func (s *SingleIndexSuite) SetupTest() {
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

	postgres.Destroy(ctx, s.pool)
	gormDB := pgtest.OpenGormDB(s.T(), source, false)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	s.store = postgres.CreateTableAndNewStore(ctx, s.pool, gormDB)
	s.indexer = postgres.NewIndexer(s.pool)
}

func (s *SingleIndexSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func getStruct(id int) *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:    fmt.Sprintf("string-%d", id),
		Name:   fmt.Sprintf("name-%d", id),
		Uint64: uint64(id),
	}
}

func (s *SingleIndexSuite) TestDocIDs() {
	var testStructs []*storage.TestSingleKeyStruct
	for i := 0; i < 8; i++ {
		testStructs = append(testStructs, getStruct(i))
	}
	s.NoError(s.store.UpsertMany(ctx, testStructs))

	for _, testCase := range []struct {
		desc   string
		docIDs []string
	}{
		{
			"none",
			[]string{},
		},
		{
			"one",
			[]string{"string-1"},
		},
		{
			"many",
			[]string{"string-1", "string-3"},
		},
	} {
		s.Run(testCase.desc, func() {
			so := search.NewSortOption(search.DocID)
			q := search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).WithPagination(search.NewPagination().AddSortOption(so)).ProtoQuery()
			results, err := s.indexer.Search(q)
			s.Require().NoError(err)
			s.Equal(testCase.docIDs, search.ResultsToIDs(results))

			q = search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).WithPagination(search.NewPagination().AddSortOption(so.Reversed(true))).ProtoQuery()
			results, err = s.indexer.Search(q)
			s.Require().NoError(err)

			sort.Sort(sort.Reverse(sort.StringSlice(testCase.docIDs)))
			s.Equal(testCase.docIDs, search.ResultsToIDs(results))
		})
	}

}

func (s *SingleIndexSuite) TestSearchAfter() {
	var testStructs []*storage.TestSingleKeyStruct
	for i := 0; i < 4; i++ {
		obj := getStruct(i)
		obj.Uint64 = uint64(i / 2)
		testStructs = append(testStructs, obj)
	}
	s.NoError(s.store.UpsertMany(ctx, testStructs))

	for _, testCase := range []struct {
		desc       string
		pagination *search.Pagination
		results    []string
		valid      bool
	}{
		{
			"none",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName)),
			[]string{"string-0", "string-1", "string-2", "string-3"},
			true,
		},
		{
			"first",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-0")),
			[]string{"string-1", "string-2", "string-3"},
			true,
		},
		{
			"first reverse",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-0").Reversed(true)),
			[]string{},
			true,
		},
		{
			"second",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-1")),
			[]string{"string-2", "string-3"},
			true,
		},
		{
			"second reverse",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-1").Reversed(true)),
			[]string{"string-0"},
			true,
		},
		{
			"two sorts",
			search.NewPagination().
				AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-0")).
				AddSortOption(search.NewSortOption(search.TestUint64).SearchAfter("0")),
			[]string{},
			false,
		},
	} {
		s.Run(testCase.desc, func() {
			q := search.NewQueryBuilder().WithPagination(testCase.pagination).ProtoQuery()
			results, err := s.indexer.Search(q)
			s.Equal(testCase.valid, err == nil)
			s.Equal(testCase.results, search.ResultsToIDs(results))
		})
	}

}
