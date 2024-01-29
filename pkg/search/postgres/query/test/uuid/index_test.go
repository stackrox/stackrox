//go:build sql_integration

package uuid

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
	pgStore "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testuuidkey/postgres"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type SingleUUIDIndexSuite struct {
	suite.Suite

	pool    postgres.DB
	store   pgStore.Store
	indexer interface {
		Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	}
}

func TestSingleUUIDIndex(t *testing.T) {
	suite.Run(t, new(SingleUUIDIndexSuite))
}

func (s *SingleUUIDIndexSuite) SetupTest() {

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

func (s *SingleUUIDIndexSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func getUUIDStruct(id int) *storage.TestSingleUUIDKeyStruct {
	return &storage.TestSingleUUIDKeyStruct{
		Key:    fmt.Sprintf("aaaaaaaa-bbbb-4011-0000-1111111111%02d", id),
		Name:   fmt.Sprintf("name-%d", id),
		Uint64: uint64(id),
	}
}

func (s *SingleUUIDIndexSuite) TestDocIDs() {
	var testStructs []*storage.TestSingleUUIDKeyStruct
	for i := 0; i < 8; i++ {
		testStructs = append(testStructs, getUUIDStruct(i))
	}
	s.NoError(s.store.UpsertMany(ctx, testStructs))

	for _, testCase := range []struct {
		desc      string
		docIDs    []string
		errorText string
	}{
		{
			"none",
			[]string{},
			"",
		},
		{
			"one",
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111101"},
			"",
		},
		{
			"many",
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111101", "aaaaaaaa-bbbb-4011-0000-111111111103"},
			"",
		},
		{
			"not a uuid",
			[]string{"junk"},
			"cannot parse UUID junk",
		},
	} {
		s.Run(testCase.desc, func() {
			so := search.NewSortOption(search.DocID)
			q := search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).WithPagination(search.NewPagination().AddSortOption(so)).ProtoQuery()
			results, err := s.indexer.Search(ctx, q)
			if testCase.errorText != "" {
				s.Error(err, testCase.errorText)
			} else {
				s.Require().NoError(err)
				s.Equal(testCase.docIDs, search.ResultsToIDs(results))

				q = search.NewQueryBuilder().AddDocIDs(testCase.docIDs...).WithPagination(search.NewPagination().AddSortOption(so.Reversed(true))).ProtoQuery()
				results, err = s.indexer.Search(ctx, q)
				s.Require().NoError(err)

				sort.Sort(sort.Reverse(sort.StringSlice(testCase.docIDs)))
				s.Equal(testCase.docIDs, search.ResultsToIDs(results))
			}
		})
	}

}

func (s *SingleUUIDIndexSuite) TestSearchAfter() {
	var testStructs []*storage.TestSingleUUIDKeyStruct
	for i := 0; i < 4; i++ {
		obj := getUUIDStruct(i)
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
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111100", "aaaaaaaa-bbbb-4011-0000-111111111101", "aaaaaaaa-bbbb-4011-0000-111111111102", "aaaaaaaa-bbbb-4011-0000-111111111103"},
			true,
		},
		{
			"first",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-0")),
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111101", "aaaaaaaa-bbbb-4011-0000-111111111102", "aaaaaaaa-bbbb-4011-0000-111111111103"},
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
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111102", "aaaaaaaa-bbbb-4011-0000-111111111103"},
			true,
		},
		{
			"second reverse",
			search.NewPagination().AddSortOption(search.NewSortOption(search.TestName).SearchAfter("name-1").Reversed(true)),
			[]string{"aaaaaaaa-bbbb-4011-0000-111111111100"},
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

func (s *SingleUUIDIndexSuite) TestAutocomplete() {
	obj := getUUIDStruct(10)
	obj.StringSlice = []string{"Hey", "Hello", "whats up"}
	obj.Uint64 = 150
	obj.Labels = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val2",
	}
	obj.Enum = storage.TestSingleUUIDKeyStruct_ENUM2
	obj.Enums = []storage.TestSingleUUIDKeyStruct_Enum{
		storage.TestSingleUUIDKeyStruct_ENUM0,
		storage.TestSingleUUIDKeyStruct_ENUM1,
	}
	s.NoError(s.store.Upsert(ctx, obj))

	optionsMap := schema.TestSingleUUIDKeyStructsSchema.OptionsMap
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

func (s *SingleUUIDIndexSuite) TestMatches() {
	testStruct0 := &storage.TestSingleUUIDKeyStruct{
		Key:    "aaaaaaaa-abcd-4011-0000-000000000000",
		Name:   "name-0",
		Uint64: 0,
	}

	testStruct1 := &storage.TestSingleUUIDKeyStruct{
		Key:    "aaaaaaaa-abab-4011-0000-111111111111",
		Name:   "name-1",
		Uint64: 1,
	}

	testStruct2 := &storage.TestSingleUUIDKeyStruct{
		Key:    "bbbbbbbb-abcd-4011-0000-111111111111",
		Name:   "name-2",
		Uint64: 2,
	}

	s.NoError(s.store.Upsert(ctx, testStruct0))
	s.NoError(s.store.Upsert(ctx, testStruct1))
	s.NoError(s.store.Upsert(ctx, testStruct2))

	for _, testCase := range []struct {
		desc            string
		q               *v1.Query
		expectedResults []*storage.TestSingleUUIDKeyStruct
		expectErr       bool
	}{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestKey, "bbbbbbbb-abcd-4011-0000-111111111111").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{testStruct2},
		},
		{
			desc:            "exact match invalid value",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestKey, "not-a-uuid").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{},
			expectErr:       true,
		},
		{
			desc:            "exact match but case insensitive on field",
			q:               search.NewQueryBuilder().AddExactMatches(search.FieldLabel("tEST kEY"), "bbbbbbbb-abcd-4011-0000-111111111111").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{testStruct2},
		},
		{
			desc:            "exact match no results",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestKey, "bbbbbbbb-0000-0000-0000-000000000000").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{},
		},
		{
			desc:            "regex",
			q:               search.NewQueryBuilder().AddRegexes(search.TestKey, "abcd.*").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            "regex no results",
			q:               search.NewQueryBuilder().AddRegexes(search.TestKey, "zzzz.*").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{},
		},
		{
			desc:            "negated",
			q:               search.NewQueryBuilder().AddStrings(search.TestKey, "!bbbbbbbb-abcd-4011-0000-111111111111").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{testStruct0, testStruct1},
		},
		{
			desc:            "negated regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestKey, "!r/abcd.*").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{testStruct1},
		},
	} {
		s.Run(testCase.desc, func() {
			results, err := s.indexer.Search(ctx, testCase.q)
			if testCase.expectErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			actualIDs := make([]string, 0, len(results))
			for _, res := range results {
				actualIDs = append(actualIDs, res.ID)
			}

			expectedIDs := make([]string, 0, len(testCase.expectedResults))
			for _, s := range testCase.expectedResults {
				expectedIDs = append(expectedIDs, s.Key)
			}
			s.ElementsMatch(actualIDs, expectedIDs)
		})
	}
}
