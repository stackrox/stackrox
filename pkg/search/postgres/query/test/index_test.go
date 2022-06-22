//go:build sql_integration
// +build sql_integration

package test

import (
	"context"
	"fmt"
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
	gormDB := pgtest.OpenGormDB(s.T(), source)
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
		Key:  fmt.Sprintf("string-%d", id),
		Name: fmt.Sprintf("name-%d", id),
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
			q := search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).ProtoQuery()
			results, err := s.indexer.Search(q)
			s.Require().NoError(err)
			s.Equal(testCase.docIDs, search.ResultsToIDs(results))
		})
	}

}
