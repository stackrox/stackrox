//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/timeutil"
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
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
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

type testCase struct {
	desc            string
	q               *v1.Query
	expectedResults []*storage.TestMultiKeyStruct
	expectErr       bool
}

func (s *IndexSuite) runTestCases(cases []testCase) {
	for _, c := range cases {
		s.Run(c.desc, func() {
			results, err := s.indexer.Search(c.q)
			if c.expectErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			actualIDs := make([]string, 0, len(results))
			for _, res := range results {
				actualIDs = append(actualIDs, res.ID)
			}

			expectedIDs := make([]string, 0, len(c.expectedResults))
			for _, s := range c.expectedResults {
				expectedIDs = append(expectedIDs, getID(s))
			}
			s.ElementsMatch(actualIDs, expectedIDs)
		})
	}
}

func (s *IndexSuite) TestString() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.String_ = "first"
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Nested: "nested_first",
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Nested2: "nested2_first",
			},
		})
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.String_ = "second"
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Nested: "nested_second",
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Nested2: "nested2_second",
			},
		})
	})
	testStruct2 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.String_ = "fir"
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Nested: "nested_fir",
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Nested2: "nested2_fir",
			},
		})
	})
	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestString, "fir").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2},
		},
		{
			desc:            "prefix",
			q:               search.NewQueryBuilder().AddStrings(search.TestString, "fir").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            "regex",
			q:               search.NewQueryBuilder().AddRegexes(search.TestString, "f.*").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            "negated prefix",
			q:               search.NewQueryBuilder().AddStrings(search.TestString, "!fir").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            "negated regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestString, "!r/.*s.*").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2},
		},
		{
			desc:            "exact match nested string",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestNestedString, "nested_second").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc: "negated prefix top-level and exact match nested string",
			q: search.NewQueryBuilder().
				AddStrings(search.TestString, "!fir").
				AddExactMatches(search.TestNestedString, "nested_second").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc: "prefix match nested string",
			q: search.NewQueryBuilder().
				AddStrings(search.TestNestedString, "nested_fir").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            "negated prefix match nested string",
			q:               search.NewQueryBuilder().AddStrings(search.TestNestedString2, "!nested2_fir").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
	})
}

func (s *IndexSuite) TestBool() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Bool = false
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			IsNested: true,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				IsNested: false,
			}})
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Bool = true
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			IsNested: false,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				IsNested: true,
			}})
	})
	s.runTestCases([]testCase{
		{
			desc:            "false",
			q:               search.NewQueryBuilder().AddBools(search.TestBool, false).ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            "true",
			q:               search.NewQueryBuilder().AddBools(search.TestBool, true).ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            "nested true",
			q:               search.NewQueryBuilder().AddBools(search.TestNestedBool, true).ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc: "nest true + false",
			q: search.NewQueryBuilder().
				AddBools(search.TestNestedBool, false).
				AddBools(search.TestNestedBool2, true).ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
	})
}

func (s *IndexSuite) TestStringSlice() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"yeah", "no"}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"whatever", "blah", "yeahyeah"}
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestStringSlice, "yeah").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
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
	})
}

func (s *IndexSuite) TestUint64() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Uint64 = 2
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Uint64 = 7
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddStrings(search.TestUint64, "2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestUint64, ">5").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            ">=",
			q:               search.NewQueryBuilder().AddStrings(search.TestUint64, ">=2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
	})
}

func (s *IndexSuite) TestInt64() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Int64 = -2
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Int64: -100,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Int64: -150,
			}})
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Int64 = 7
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Int64: 100,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Int64: -200,
			}})
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64, "-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64, ">5").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            ">=",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64, ">=-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
		{
			desc:            "nested",
			q:               search.NewQueryBuilder().AddStrings(search.TestNestedInt64, "<-50").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc: "nested and nested2",
			q: search.NewQueryBuilder().AddStrings(search.TestNestedInt64, ">=0").
				AddStrings(search.TestNested2Int64, ">=-200").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
	})
}

func (s *IndexSuite) TestIntArray() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.IntSlice = []int64{-2, 5}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.IntSlice = []int64{7, 3}
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64Slice, "-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64Slice, ">5").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            ">=",
			q:               search.NewQueryBuilder().AddStrings(search.TestInt64Slice, ">=-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
	})
}

func (s *IndexSuite) TestFloat() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Float = -2
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Float = 7.5
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddStrings(search.TestFloat, "-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestFloat, ">7.3").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            ">=",
			q:               search.NewQueryBuilder().AddStrings(search.TestFloat, ">=-2").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
	})
}

func (s *IndexSuite) TestMap() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Labels = map[string]string{
			"foo": "bar",
			"new": "old",
		}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Labels = map[string]string{
			"one":   "two",
			"three": "four",
		}
	})

	testStruct2 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
	})

	s.runTestCases([]testCase{
		{
			desc:            "key exists",
			q:               search.NewQueryBuilder().AddMapQuery(search.TestLabels, "foo", "").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc:            "key does not exist",
			q:               search.NewQueryBuilder().AddMapQuery(search.TestLabels, "!foo", "").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1, testStruct2},
		},
		{
			desc:      "negated key and value, should get error",
			q:         search.NewQueryBuilder().AddMapQuery(search.TestLabels, "!foo", "blah").ProtoQuery(),
			expectErr: true,
		},
		{
			desc:            "non-empty map",
			q:               search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
		{
			desc:            "value only",
			q:               search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "bar").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
		{
			desc: "negated value only",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "!bar").ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
		{
			desc: "key and negated value, doesn't match",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "foo", "!bar").ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: []*storage.TestMultiKeyStruct{},
		},
		{
			desc: "key and negated value, matches",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "foo", "!r/c.*").ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0},
		},
	})
}

var (
	ts2029Mar09Noon = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2029-03-09T12:00:00Z"))
	ts2022Mar09Noon = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2022-03-09T12:00:00Z"))
	ts2022Feb09Noon = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2022-02-09T12:00:00Z"))
	ts2021Mar09Noon = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2021-03-09T12:00:00Z"))
	ts2020Mar09Noon = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2020-03-09T12:00:00Z"))
)

func (s *IndexSuite) TestTime() {
	testStruct2029Mar09Noon := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2029Mar09Noon
	})
	testStruct2022Mar09Noon := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2022Mar09Noon
	})
	testStruct2022Feb09Noon := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2022Feb09Noon
	})
	testStruct2021Mar09Noon := s.getStruct(3, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2021Mar09Noon
	})
	testStruct2020Mar09Noon := s.getStruct(4, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2020Mar09Noon
	})
	_ = testStruct2029Mar09Noon

	s.runTestCases([]testCase{
		{
			desc:            "exact match (should evaluate if it's within the day) - matches",
			q:               search.NewQueryBuilder().AddStrings(search.TestTimestamp, "03/09/2022 UTC").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2022Mar09Noon},
		},
		{
			desc:            "exact match (should evaluate if it's within the day) - no match",
			q:               search.NewQueryBuilder().AddStrings(search.TestTimestamp, "03/08/2022").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{},
		},
		{
			desc:            "< date",
			q:               search.NewQueryBuilder().AddStrings(search.TestTimestamp, "< 03/09/2022").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2021Mar09Noon, testStruct2020Mar09Noon, testStruct2022Feb09Noon},
		},
		{
			desc:            "< date time (this time, includes Mar 10th at noon)",
			q:               search.NewQueryBuilder().AddStrings(search.TestTimestamp, "< 03/09/2022 1:00 PM").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2021Mar09Noon, testStruct2020Mar09Noon, testStruct2022Feb09Noon, testStruct2022Mar09Noon},
		},
		{
			desc:            "> duration (this test will fail in 2029, but hopefully it's not still being run then)",
			q:               search.NewQueryBuilder().AddStrings(search.TestTimestamp, "> 1d").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2021Mar09Noon, testStruct2020Mar09Noon, testStruct2022Feb09Noon, testStruct2022Mar09Noon},
		},
	})
}

func (s *IndexSuite) TestEnum() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM0
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM1
	})
	testStruct2 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM2
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestEnum, "ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            "negation",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnum, "!ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            "regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnum, "r/E.*1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1},
		},
		{
			desc:            "negated regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnum, "!r/E.*1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct2},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnum, ">ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct2},
		},
		{
			desc:            "<=",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnum, "<=ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1},
		},
	})
}

func (s *IndexSuite) TestEnumArray() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM1}
	})
	testStruct01 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0, storage.TestMultiKeyStruct_ENUM1}
	})
	testStruct012 := s.getStruct(3, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0, storage.TestMultiKeyStruct_ENUM1, storage.TestMultiKeyStruct_ENUM2}
	})
	testStruct12 := s.getStruct(4, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM1, storage.TestMultiKeyStruct_ENUM2}
	})

	s.runTestCases([]testCase{
		{
			desc:            "exact match",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestEnumSlice, "ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1, testStruct01, testStruct012, testStruct12},
		},
		{
			desc:            "negation",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnumSlice, "!ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct01, testStruct012, testStruct12},
		},
		{
			desc:            "regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnumSlice, "r/E.*1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct1, testStruct01, testStruct012, testStruct12},
		},
		{
			desc:            "negated regex",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnumSlice, "!r/E.*1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct01, testStruct012, testStruct12},
		},
		{
			desc:            ">",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnumSlice, ">ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct012, testStruct12},
		},
		{
			desc:            "<=",
			q:               search.NewQueryBuilder().AddStrings(search.TestEnumSlice, "<=ENUM1").ProtoQuery(),
			expectedResults: []*storage.TestMultiKeyStruct{testStruct0, testStruct1, testStruct12, testStruct012, testStruct01},
		},
	})
}

type highlightTestCase struct {
	desc            string
	q               *v1.Query
	expectedResults map[*storage.TestMultiKeyStruct]map[string][]string
	expectErr       bool
}

func (s *IndexSuite) runHighlightTestCases(cases []highlightTestCase) {
	for _, c := range cases {
		s.Run(c.desc, func() {
			results, err := s.indexer.Search(c.q)
			if c.expectErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			actualResults := make(map[string]map[string][]string, len(results))
			for _, res := range results {
				actualResults[res.ID] = res.Matches
			}

			for obj, expectedMatches := range c.expectedResults {
				id := getID(obj)
				matchingActual, ok := actualResults[id]
				if !s.True(ok, "id %s expected but not found", id) {
					continue
				}
				s.Equal(expectedMatches, matchingActual, "mismatch for id %s", id)
				delete(actualResults, id)
			}
			s.Empty(actualResults, "Unexpected results found: %+v", actualResults)
		})
	}
}

func (s *IndexSuite) TestStringHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.String_ = "zero"
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.String_ = "one"
	})
	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "prefix query, one match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestString, "ze").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.string": {"zero"},
				},
			},
		},
		{
			desc: "regex query, two matches",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestString, "r/.*o.*").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.string": {"zero"},
				},
				testStruct1: {
					"multi.string": {"one"},
				},
			},
		},
	})
}

func (s *IndexSuite) TestBoolHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Bool = false
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Bool = true
	})
	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "true query",
			q:    search.NewQueryBuilder().AddBoolsHighlighted(search.TestBool, true).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {
					"multi.bool": {"true"},
				},
			},
		},
		{
			desc: "false query",
			q:    search.NewQueryBuilder().AddBoolsHighlighted(search.TestBool, false).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.bool": {"false"},
				},
			},
		},
		{
			desc: "true or false query",
			q:    search.NewQueryBuilder().AddBoolsHighlighted(search.TestBool, true, false).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.bool": {"false"},
				},
				testStruct1: {
					"multi.bool": {"true"},
				},
			},
		},
	})
}

func (s *IndexSuite) TestStringSliceHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"yeah", "no"}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.StringSlice = []string{"whatever", "blah", "yeahyeah"}
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestStringSlice, `"yeah"`).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.string_slice": {"yeah"},
				},
			},
		},
		{
			desc: "regex",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestStringSlice, "r/.*e.*").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {
					"multi.string_slice": {"yeah"},
				},
				testStruct1: {
					"multi.string_slice": {"whatever", "yeahyeah"},
				},
			},
		},
	})
}

func (s *IndexSuite) TestTimeHighlights() {
	testStruct2029Mar09Noon := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2029Mar09Noon
	})
	testStruct2022Mar09Noon := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2022Mar09Noon
	})
	testStruct2022Feb09Noon := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2022Feb09Noon
	})
	testStruct2021Mar09Noon := s.getStruct(3, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2021Mar09Noon
	})
	testStruct2020Mar09Noon := s.getStruct(4, func(s *storage.TestMultiKeyStruct) {
		s.Timestamp = ts2020Mar09Noon
	})
	_ = testStruct2029Mar09Noon

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match (should evaluate if it's within the day) - matches",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestTimestamp, "03/09/2022 UTC").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct2022Mar09Noon: {"multi.timestamp.seconds": {"2022-03-09 12:00:00"}},
			},
		},
		{
			desc:            "exact match (should evaluate if it's within the day) - no match",
			q:               search.NewQueryBuilder().AddStringsHighlighted(search.TestTimestamp, "03/08/2022").ProtoQuery(),
			expectedResults: nil,
		},
		{
			desc: "< date",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestTimestamp, "< 03/09/2022").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct2021Mar09Noon: {"multi.timestamp.seconds": {"2021-03-09 12:00:00"}},
				testStruct2020Mar09Noon: {"multi.timestamp.seconds": {"2020-03-09 12:00:00"}},
				testStruct2022Feb09Noon: {"multi.timestamp.seconds": {"2022-02-09 12:00:00"}},
			},
		},
		{
			desc: "< date time (this time, includes Mar 10th at noon)",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestTimestamp, "< 03/09/2022 1:00 PM").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct2021Mar09Noon: {"multi.timestamp.seconds": {"2021-03-09 12:00:00"}},
				testStruct2020Mar09Noon: {"multi.timestamp.seconds": {"2020-03-09 12:00:00"}},
				testStruct2022Feb09Noon: {"multi.timestamp.seconds": {"2022-02-09 12:00:00"}},
				testStruct2022Mar09Noon: {"multi.timestamp.seconds": {"2022-03-09 12:00:00"}},
			},
		},
		{
			desc: "> duration (this test will fail in 2029, but hopefully it's not still being run then)",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestTimestamp, "> 1d").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct2021Mar09Noon: {"multi.timestamp.seconds": {"2021-03-09 12:00:00"}},
				testStruct2020Mar09Noon: {"multi.timestamp.seconds": {"2020-03-09 12:00:00"}},
				testStruct2022Feb09Noon: {"multi.timestamp.seconds": {"2022-02-09 12:00:00"}},
				testStruct2022Mar09Noon: {"multi.timestamp.seconds": {"2022-03-09 12:00:00"}},
			},
		},
	})
}

func (s *IndexSuite) TestEnumHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM0
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM1
	})
	testStruct2 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Enum = storage.TestMultiKeyStruct_ENUM2
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, `"ENUM1"`).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.enum": {"ENUM1"}},
			},
		},
		{
			desc: "negation",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, "!ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.enum": {"ENUM0"}},
				testStruct2: {"multi.enum": {"ENUM2"}},
			},
		},
		{
			desc: "regex",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, "r/E.*1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.enum": {"ENUM1"}},
			},
		},
		{
			desc: "negated regex",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, "!r/E.*1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.enum": {"ENUM0"}},
				testStruct2: {"multi.enum": {"ENUM2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, ">ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct2: {"multi.enum": {"ENUM2"}},
			},
		},
		{
			desc: "<=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnum, "<=ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.enum": {"ENUM0"}},
				testStruct1: {"multi.enum": {"ENUM1"}},
			},
		},
	})
}

func (s *IndexSuite) TestUint64Highlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Uint64 = 2
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Uint64 = 7
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestUint64, "2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.uint64": {"2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestUint64, ">5").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.uint64": {"7"}},
			},
		},
		{
			desc: ">=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestUint64, ">=2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.uint64": {"2"}},
				testStruct1: {"multi.uint64": {"7"}},
			},
		},
	})
}

func (s *IndexSuite) TestInt64Highlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Int64 = -2
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Int64: -100,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Int64: -150,
			}})
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Int64 = 7
		s.Nested = append(s.Nested, &storage.TestMultiKeyStruct_Nested{
			Int64: 100,
			Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
				Int64: -200,
			}})
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64, "-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.int64": {"-2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64, ">5").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.int64": {"7"}},
			},
		},
		{
			desc: ">=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64, ">=-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.int64": {"-2"}},
				testStruct1: {"multi.int64": {"7"}},
			},
		},
		{
			desc: "nested",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestNestedInt64, "<-50").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.nested.int64": {"-100"}},
			},
		},
		{
			desc: "nested and nested2",
			q: search.NewQueryBuilder().AddStringsHighlighted(search.TestNestedInt64, ">=0").
				AddStringsHighlighted(search.TestNested2Int64, ">=-200").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.nested.nested2.int64": {"-200"}, "multi.nested.int64": {"100"}},
			},
		},
	})
}

func (s *IndexSuite) TestFloatHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Float = -2
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Float = 7.5
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestFloat, "-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.float": {"-2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestFloat, ">7.3").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.float": {"7.5"}},
			},
		},
		{
			desc: ">=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestFloat, ">=-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.float": {"-2"}},
				testStruct1: {"multi.float": {"7.5"}},
			},
		},
	})
}

func (s *IndexSuite) TestIntArrayHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.IntSlice = []int64{-2, -5}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.IntSlice = []int64{7, 3}
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64Slice, "-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.int_slice": {"-2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64Slice, ">5").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1: {"multi.int_slice": {"7"}},
			},
		},
		{
			desc: ">=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestInt64Slice, ">=-2").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.int_slice": {"-2"}},
				testStruct1: {"multi.int_slice": {"7", "3"}},
			},
		},
	})
}

func (s *IndexSuite) TestEnumArrayHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM1}
	})
	testStruct01 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0, storage.TestMultiKeyStruct_ENUM1}
	})
	testStruct012 := s.getStruct(3, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM0, storage.TestMultiKeyStruct_ENUM1, storage.TestMultiKeyStruct_ENUM2}
	})
	testStruct12 := s.getStruct(4, func(s *storage.TestMultiKeyStruct) {
		s.Enums = []storage.TestMultiKeyStruct_Enum{storage.TestMultiKeyStruct_ENUM1, storage.TestMultiKeyStruct_ENUM2}
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "exact match",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, `"ENUM1"`).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1:   {"multi.enums": {"ENUM1"}},
				testStruct01:  {"multi.enums": {"ENUM1"}},
				testStruct012: {"multi.enums": {"ENUM1"}},
				testStruct12:  {"multi.enums": {"ENUM1"}},
			},
		},
		{
			desc: "negation",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, "!ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0:   {"multi.enums": {"ENUM0"}},
				testStruct01:  {"multi.enums": {"ENUM0"}},
				testStruct012: {"multi.enums": {"ENUM0", "ENUM2"}},
				testStruct12:  {"multi.enums": {"ENUM2"}},
			},
		},
		{
			desc: "regex",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, "r/E.*1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct1:   {"multi.enums": {"ENUM1"}},
				testStruct01:  {"multi.enums": {"ENUM1"}},
				testStruct012: {"multi.enums": {"ENUM1"}},
				testStruct12:  {"multi.enums": {"ENUM1"}},
			},
		},
		{
			desc: "negated regex",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, "!r/E.*1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0:   {"multi.enums": {"ENUM0"}},
				testStruct01:  {"multi.enums": {"ENUM0"}},
				testStruct012: {"multi.enums": {"ENUM0", "ENUM2"}},
				testStruct12:  {"multi.enums": {"ENUM2"}},
			},
		},
		{
			desc: ">",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, ">ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct012: {"multi.enums": {"ENUM2"}},
				testStruct12:  {"multi.enums": {"ENUM2"}},
			},
		},
		{
			desc: "<=",
			q:    search.NewQueryBuilder().AddStringsHighlighted(search.TestEnumSlice, "<=ENUM1").ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0:   {"multi.enums": {"ENUM0"}},
				testStruct1:   {"multi.enums": {"ENUM1"}},
				testStruct12:  {"multi.enums": {"ENUM1"}},
				testStruct012: {"multi.enums": {"ENUM0", "ENUM1"}},
				testStruct01:  {"multi.enums": {"ENUM0", "ENUM1"}},
			},
		},
	})
}

func (s *IndexSuite) TestMapHighlights() {
	testStruct0 := s.getStruct(0, func(s *storage.TestMultiKeyStruct) {
		s.Labels = map[string]string{
			"foo": "bar",
			"new": "old",
		}
	})
	testStruct1 := s.getStruct(1, func(s *storage.TestMultiKeyStruct) {
		s.Labels = map[string]string{
			"one":   "two",
			"three": "four",
			"foo":   "car",
		}
	})

	testStruct2 := s.getStruct(2, func(s *storage.TestMultiKeyStruct) {
	})

	s.runHighlightTestCases([]highlightTestCase{
		{
			desc: "key exists",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "new", "").MarkHighlighted(search.TestLabels).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.labels": {"new=old"}},
			},
		},
		{
			desc: "key does not exist",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "!foo", "").MarkHighlighted(search.TestLabels).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				// No labels will be printed since it's a "does not exist" query
				testStruct2: {},
			},
		},
		{
			desc:      "negated key and value, should get error",
			q:         search.NewQueryBuilder().AddMapQuery(search.TestLabels, "!foo", "blah").MarkHighlighted(search.TestLabels).ProtoQuery(),
			expectErr: true,
		},
		{
			desc: "non-empty map",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "").MarkHighlighted(search.TestLabels).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.labels": {"foo=bar", "new=old"}},
				testStruct1: {"multi.labels": {"foo=car", "one=two", "three=four"}},
			},
		},
		{
			desc: "value only",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "bar").MarkHighlighted(search.TestLabels).ProtoQuery(),
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.labels": {"foo=bar"}},
			},
		},
		{
			desc: "negated value only",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "", "!bar").MarkHighlighted(search.TestLabels).ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.labels": {"new=old"}},
				testStruct1: {"multi.labels": {"foo=car", "one=two", "three=four"}},
			},
		},
		{
			desc: "key and negated value, doesn't match",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "foo", "!r/.*ar").MarkHighlighted(search.TestLabels).ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{},
		},
		{
			desc: "key and negated value, matches",
			q:    search.NewQueryBuilder().AddMapQuery(search.TestLabels, "foo", "!r/c.*").MarkHighlighted(search.TestLabels).ProtoQuery(),
			// Negated value does not mean non-existence of value, it just means there should be at least one element
			// not matching the value. Unclear what the use-case of this is, but it is supported...
			expectedResults: map[*storage.TestMultiKeyStruct]map[string][]string{
				testStruct0: {"multi.labels": {"foo=bar"}},
			},
		},
	})
}
