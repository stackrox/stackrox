//go:build sql_integration

package test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgStore "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/test/postgres"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type SingleIndexSuite struct {
	suite.Suite

	pool    postgres.DB
	store   pgStore.Store
	indexer interface {
		Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	}
}

func TestSingleIndex(t *testing.T) {
	suite.Run(t, new(SingleIndexSuite))
}

func (s *SingleIndexSuite) SetupTest() {

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(context.Background(), config)
	s.Require().NoError(err)

	pgStore.Destroy(ctx, s.pool)
	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	s.store = pgStore.CreateTableAndNewStore(ctx, s.pool, gormDB)
	s.indexer = pgStore.NewIndexer(s.pool)
}

func (s *SingleIndexSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
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
			results, err := s.indexer.Search(ctx, q)
			s.Require().NoError(err)
			s.Equal(testCase.docIDs, search.ResultsToIDs(results))

			q = search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).WithPagination(search.NewPagination().AddSortOption(so.Reversed(true))).ProtoQuery()
			results, err = s.indexer.Search(ctx, q)
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
			results, err := s.indexer.Search(ctx, q)
			s.Equal(testCase.valid, err == nil)
			s.Equal(testCase.results, search.ResultsToIDs(results))
		})
	}
}

func (s *SingleIndexSuite) TestAutocomplete() {
	obj := getStruct(10)
	obj.StringSlice = []string{"Hey", "Hello", "whats up"}
	obj.Uint64 = 150
	obj.Labels = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val2",
	}
	obj.Enum = storage.TestSingleKeyStruct_ENUM2
	obj.Enums = []storage.TestSingleKeyStruct_Enum{
		storage.TestSingleKeyStruct_ENUM0,
		storage.TestSingleKeyStruct_ENUM1,
	}
	s.NoError(s.store.Upsert(ctx, obj))

	optionsMap := schema.TestSingleKeyStructsSchema.OptionsMap
	for _, testCase := range []struct {
		field       search.FieldLabel
		queryString string
		regexQuery  bool
		results     []string
	}{
		{
			field:       search.TestName,
			queryString: search.WildcardString,
			regexQuery:  false,
			results:     []string{"name-10"},
		},
		{
			field:       search.TestName,
			queryString: "name-",
			regexQuery:  true,
			results:     []string{"name-10"},
		},
		{
			field:       search.TestName,
			queryString: "nope",
			regexQuery:  false,
		},
		{
			field:       search.TestStringSlice,
			queryString: search.WildcardString,
			regexQuery:  false,
			results:     []string{"Hey", "Hello", "whats up"},
		},
		{
			field:       search.TestStringSlice,
			queryString: "He.*",
			regexQuery:  true,
			results:     []string{"Hey", "Hello"},
		},
		{
			field:       search.TestStringSlice,
			queryString: "Hello",
			regexQuery:  true,
			results:     []string{"Hello"},
		},
		{
			field:       search.TestStringSlice,
			queryString: "nope",
			regexQuery:  false,
		},
		{
			field:       search.TestEnum,
			queryString: search.WildcardString,
			regexQuery:  false,
			results:     []string{"ENUM2"},
		},
		{
			field:       search.TestEnum,
			queryString: "ENUM",
			regexQuery:  false,
			results:     []string{"ENUM2"},
		},
		{
			field:       search.TestEnum,
			queryString: "nope",
			regexQuery:  false,
		},
		{
			field:       search.TestEnumSlice,
			queryString: search.WildcardString,
			regexQuery:  false,
			results:     []string{"ENUM0", "ENUM1"},
		},
		{
			field:       search.TestEnumSlice,
			queryString: "ENUM",
			regexQuery:  false,
			results:     []string{"ENUM0", "ENUM1"},
		},
		{
			field:       search.TestEnumSlice,
			queryString: "ENUM1",
			regexQuery:  false,
			results:     []string{"ENUM1"},
		},
		{
			field:       search.TestEnumSlice,
			queryString: "no",
			regexQuery:  false,
		},
		{
			field:       search.TestLabels,
			queryString: search.WildcardString,
			regexQuery:  false,
			results:     []string{"key1=val1", "key2=val2", "key3=val2"},
		},
		{
			field:       search.TestLabels,
			queryString: "key",
			regexQuery:  false,
			results:     []string{"key1=val1", "key2=val2", "key3=val2"},
		},
		{
			field:       search.TestLabels,
			queryString: "key1",
			regexQuery:  false,
			results:     []string{"key1=val1"},
		},
		{
			field:       search.TestLabels,
			queryString: "=val",
			regexQuery:  false,
			results:     []string{"key1=val1", "key2=val2", "key3=val2"},
		},
		{
			field:       search.TestLabels,
			queryString: "=val1",
			regexQuery:  false,
			results:     []string{"key1=val1"},
		},
	} {
		s.Run(fmt.Sprintf("%s-%s", testCase.field, testCase.queryString), func() {
			qb := search.NewQueryBuilder()
			if testCase.regexQuery {
				qb.AddRegexesHighlighted(testCase.field, testCase.queryString)
			} else {
				qb.AddStringsHighlighted(testCase.field, testCase.queryString)
			}
			results, err := s.indexer.Search(ctx, qb.ProtoQuery())
			s.NoError(err)
			if len(testCase.results) > 0 {
				s.Require().Len(results, 1)

				field := optionsMap.MustGet(testCase.field.String())
				s.ElementsMatch(testCase.results, results[0].Matches[field.FieldPath])
			} else {
				s.Len(results, 0)
			}
		})
	}
}
