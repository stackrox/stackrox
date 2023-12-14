//go:build sql_integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type selectQTestCase struct {
	desc           string
	q              *v1.Query
	resultStruct   any
	expectedError  string
	expectedQuery  string
	expectedResult any
}

type Struct1 struct {
	TestString string `db:"test_string"`
}

type Struct2 struct {
	TestNestedString string `db:"test_nested_string"`
	TestNestedBool   bool   `db:"test_nested_bool"`
}

type Struct2GrpBy1 struct {
	TestNestedString []string `db:"test_nested_string"`
	TestNestedBool   []bool   `db:"test_nested_bool"`
	TestString       string   `db:"test_string"`
}

type Struct2GrpBy2 struct {
	TestNestedString []string `db:"test_nested_string"`
	TestNestedBool   []bool   `db:"test_nested_bool"`
	TestString       string   `db:"test_string"`
	TestBool         bool     `db:"test_bool"`
}

type Struct2GrpBy3 struct {
	TestTimestamp []*time.Time `db:"test_timestamp"`
	TestString    string       `db:"test_string"`
}

type Struct2GrpBy4 struct {
	TestTimestamp *time.Time `db:"test_timestamp_max"`
	TestString    string     `db:"test_string"`
}

type Struct2GrpBy5 struct {
	TestTimestamp time.Time `db:"test_timestamp_max"`
	TestString    string    `db:"test_string"`
}

type Struct3 struct {
	TestString       string `db:"test_string"`
	TestNestedString string `db:"test_nested_string"`
}

type Struct4 struct {
	TestString       string   `db:"test_string"`
	TestNestedString []string `db:"test_nested_string"`
}

type Struct5 struct {
	TestString    string     `db:"test_string"`
	TestTimestamp *time.Time `db:"test_timestamp"`
}

func TestSelectQuery(t *testing.T) {
	t.Parallel()

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	defer testDB.Teardown(t)

	store := postgres.New(testDB.DB)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	for _, c := range []selectQTestCase{
		{
			desc: "base schema; no select",
			q: search.NewQueryBuilder().
				AddExactMatches(search.TestString, "acs").ProtoQuery(),
			resultStruct:   Struct1{},
			expectedError:  "select portion of the query cannot be empty",
			expectedResult: nil,
		},
		{
			desc: "base schema; select null timestamp",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestString), search.NewQuerySelect(search.TestTimestamp),
				).ProtoQuery(),
			resultStruct:  Struct5{},
			expectedQuery: "select test_structs.String_, test_structs.timestamp as test_timestamp from test_structs",
			expectedResult: []*Struct5{
				{"acs", nil},
				{"acs", nil},
				{"bcs", nil},
				{"bcs", nil},
			},
		},
		{
			desc: "base schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.TestString)).ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_structs.String_ as test_string from test_structs",
			expectedResult: []*Struct1{
				{"acs"},
				{"acs"},
				{"bcs"},
				{"bcs"},
			},
		},
		{
			desc: "base schema; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.TestString)).
				AddExactMatches(search.TestString, "acs").ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_structs.String_ as test_string from test_structs where test_structs.String_ = $1",
			expectedResult: []*Struct1{
				{"acs"},
				{"acs"},
			},
		},
		{
			desc: "child schema; multiple select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString),
					search.NewQuerySelect(search.TestNestedBool),
				).
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct2{},
			expectedQuery: "select test_structs_nesteds.Nested as test_nested_string, test_structs_nesteds.IsNested as test_nested_bool " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs_nesteds.Nested = $1",
			expectedResult: []*Struct2{
				{
					TestNestedString: "nested_acs",
					TestNestedBool:   false,
				},
			},
		},
		{
			desc: "child schema; multiple select w/ where & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString),
					search.NewQuerySelect(search.TestNestedBool),
				).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_structs_nesteds.Nested) as test_nested_string, " +
				"jsonb_agg(test_structs_nesteds.IsNested) as test_nested_bool, test_structs.String_ as test_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs_nesteds.Nested = $1 " +
				"group by test_structs.String_",
			expectedResult: []*Struct2GrpBy1{
				{
					TestNestedString: []string{"nested_acs"},
					TestNestedBool:   []bool{false},
					TestString:       "acs",
				},
			},
		},
		{
			desc: "child schema; multiple select & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString),
					search.NewQuerySelect(search.TestNestedBool),
				).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_structs_nesteds.Nested) as test_nested_string, jsonb_agg(test_structs_nesteds.IsNested) as test_nested_bool, test_structs.String_ as test_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs.String_",
			expectedResult: []*Struct2GrpBy1{
				{
					TestNestedString: []string{"nested_acs"},
					TestNestedBool:   []bool{false},
					TestString:       "acs",
				},
				{
					TestNestedString: []string{"nested_bcs_1", "nested_bcs_1", "nested_bcs_2"},
					TestNestedBool:   []bool{false, false, false},
					TestString:       "bcs",
				},
			},
		},
		{
			desc: "select null timestamp & group by; scan into pointer",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestTimestamp),
				).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy3{},
			expectedQuery: "select jsonb_agg(test_structs.timestamp) as test_timestamp " +
				"from test_structs " +
				"group by test_structs.String_",
			expectedResult: []*Struct2GrpBy3{
				{
					TestTimestamp: []*time.Time{nil, nil},
					TestString:    "acs",
				},
				{
					TestTimestamp: []*time.Time{nil, nil},
					TestString:    "bcs",
				},
			},
		},
		{
			desc: "select max null timestamp & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestTimestamp).AggrFunc(aggregatefunc.Max),
				).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy4{},
			expectedQuery: "select max(test_structs.timestamp) as test_timestamp_max " +
				"from test_structs " +
				"group by test_structs.String_",
			expectedResult: []*Struct2GrpBy4{
				{
					TestTimestamp: nil,
					TestString:    "acs",
				},
				{
					TestTimestamp: nil,
					TestString:    "bcs",
				},
			},
		},
		{
			desc: "select max null timestamp & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestTimestamp).AggrFunc(aggregatefunc.Max),
				).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy5{},
			expectedQuery: "select max(test_structs.timestamp) as test_timestamp_max " +
				"from test_structs " +
				"group by test_structs.String_",
			expectedError: "cannot assign NULL to *time.Time",
		},
		{
			desc: "child schema; multiple select w/ where & multiple group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString),
					search.NewQuerySelect(search.TestNestedBool),
				).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString, search.TestBool).ProtoQuery(),
			resultStruct: Struct2GrpBy2{},
			expectedQuery: "select jsonb_agg(test_structs_nesteds.Nested) as test_nested_string, " +
				"jsonb_agg(test_structs_nesteds.IsNested) as test_nested_bool, " +
				"test_structs.String_ as test_string, test_structs.Bool as test_bool " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs_nesteds.Nested = $1 " +
				"group by test_structs.String_, test_structs.Bool",
			expectedResult: []*Struct2GrpBy2{
				{
					TestNestedString: []string{"nested_acs"},
					TestNestedBool:   []bool{false},
					TestString:       "acs",
					TestBool:         true,
				},
			},
		},
		{
			desc: "base schema and child schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestString),
					search.NewQuerySelect(search.TestNestedString),
				).ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_structs.String_ as test_string, test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*Struct3{
				{
					TestString:       "acs",
					TestNestedString: "nested_acs",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_1",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_1",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_2",
				},
			},
		},
		{
			desc: "base schema and child schema conjunction query; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestString),
					search.NewQuerySelect(search.TestNestedString),
				).
				AddExactMatches(search.TestString, "acs").
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_structs.String_ as test_string, test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where (test_structs_nesteds.Nested = $1 and test_structs.String_ = $2)",
			expectedResult: []*Struct3{
				{
					TestString:       "acs",
					TestNestedString: "nested_acs",
				},
			},
		},
		{
			desc: "base schema and child schema; select; pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestString),
					search.NewQuerySelect(search.TestNestedString),
				).WithPagination(
				search.NewPagination().
					AddSortOption(search.NewSortOption(search.TestString)).
					AddSortOption(search.NewSortOption(search.TestNestedString)),
			).ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_structs.String_ as test_string, test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"order by test_structs.String_ asc, test_structs_nesteds.Nested as asc",
			expectedResult: []*Struct3{
				{
					TestString:       "acs",
					TestNestedString: "nested_acs",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_1",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_1",
				},
				{
					TestString:       "bcs",
					TestNestedString: "nested_bcs_2",
				},
			},
		},
		{
			desc: "base schema and child schema; select; pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestString),
					search.NewQuerySelect(search.TestNestedString),
				).
				AddGroupBy(search.TestString).
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.TestString).Reversed(true)),
				).ProtoQuery(),
			resultStruct: Struct4{},
			expectedQuery: "select test_structs.String_ as test_string, json_agg(test_structs_nesteds.Nested) as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs.String_" +
				"order by test_structs.String_ desc",
			expectedResult: []*Struct4{
				{
					TestString:       "bcs",
					TestNestedString: []string{"nested_bcs_1", "nested_bcs_1", "nested_bcs_2"},
				},
				{
					TestString:       "acs",
					TestNestedString: []string{"nested_acs"},
				},
			},
		},
		{
			desc: "nil query",
			q:    nil,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			runTest(ctx, t, testDB, c)
		})
	}
}

type DerivedStruct1 struct {
	TestNestedStringCount int `db:"test_nested_string_count"`
}

type DerivedStruct2 struct {
	TestNestedStringCount  int `db:"test_nested_string_count"`
	TestNestedString2Count int `db:"test_nested_string_2_count"`
}

type DerivedStruct22 struct {
	TestNestedStringCount int    `db:"test_nested_string_count"`
	TopTestNestedString2  string `db:"test_nested_string_2_max"`
}

type DerivedStruct3 struct {
	TestNestedStringCount  int    `db:"test_nested_string_count"`
	TestNestedString2Count int    `db:"test_nested_string_2_count"`
	TestNestedString       string `db:"test_nested_string"`
}

type DerivedStruct4 struct {
	TestNestedStringCount int    `db:"test_nested_string_count"`
	TestString            string `db:"test_string"`
}

type DerivedStruct5 struct {
	TestNestedStringCount int    `db:"test_nested_string_count"`
	TestNestedString      string `db:"test_nested_string"`
}

type DerivedStruct6 struct {
	TestNestedStringCount int      `db:"test_nested_string_count"`
	TestString            []string `db:"test_string"`
	TestNestedString      string   `db:"test_nested_string"`
}

type DerivedStruct7 struct {
	TestStringCountWithEnum1 int `db:"test_string_affected_by_enum1"`
	TestStringCountWithEnum2 int `db:"test_string_affected_by_enum2"`
}

type DerivedStruct8 struct {
	TestStringAffectedByEnum1 int  `db:"test_string_affected_by_enum1"`
	TestStringAffectedByEnum2 int  `db:"test_string_affected_by_enum2"`
	TestBool                  bool `db:"test_bool"`
}

type DerivedStruct9 struct {
	TestNestedStringCount int      `db:"test_nested_string_count"`
	TestString            []string `db:"test_string"`
	TestStringCount       int      `db:"test_string_count"`
	TestNestedString      string   `db:"test_nested_string"`
}

func TestSelectDerivedFieldQuery(t *testing.T) {
	t.Parallel()

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	defer testDB.Teardown(t)

	store := postgres.New(testDB.DB)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	for _, c := range []selectQTestCase{
		{
			desc: "select one derived",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
				).ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct1{
				{4},
			},
		},
		{
			desc: "select one derived w/ distinct",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count).Distinct(),
				).ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(distinct(test_structs_nesteds.Nested)) as test_nested_string_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct1{
				{3},
			},
		},
		{
			desc: "select multiple derived",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count).Distinct(),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count).Distinct(),
				).ProtoQuery(),
			resultStruct: DerivedStruct2{},
			expectedQuery: "select count(distinct(test_structs_nesteds.Nested)) as test_nested_string_count, " +
				"count(distinct(test_structs_nesteds.Nested2_Nested2)) as test_nested_string_2_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct2{
				{3, 4},
			},
		},
		{
			desc: "select multiple derived again",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Max),
				).ProtoQuery(),
			resultStruct: DerivedStruct22{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"max(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_max " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct22{
				{4, "nested_bcs_nested_2"},
			},
		},
		{
			desc: "select multiple derived w/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count),
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"count(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct3{
				{1, 1, "nested_acs"},
				{2, 2, "nested_bcs_1"},
				{1, 1, "nested_bcs_2"},
			},
		},
		{
			desc: "select one derived w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count)).
				AddExactMatches(search.TestString, "bcs").ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs.String_ = $1",
			expectedResult: []*DerivedStruct1{
				{3},
			},
		},
		{
			desc: "select multiple derived w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count),
				).
				AddStrings(search.TestNestedString2, "nested").ProtoQuery(),
			resultStruct: DerivedStruct2{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"count(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs_nesteds.Nested2_Nested2 ilike $1",
			expectedResult: []*DerivedStruct2{
				{3, 3},
			},
		},
		{
			desc: "select multiple derived w/ where & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count),
				).
				AddStrings(search.TestNestedString2, "nested").
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"count(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"where test_structs_nesteds.Nested2_Nested2 ilike $1 " +
				"group by test_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct3{
				{1, 1, "nested_acs"},
				{1, 1, "nested_bcs_1"},
				{1, 1, "nested_bcs_2"},
			},
		},
		{
			desc: "select derived & primary key",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestString),
				).ProtoQuery(),
			resultStruct: DerivedStruct4{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"test_structs.String_ as test_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct4{
				{1, "acs"},
				{3, "bcs"},
			},
			expectedError: "ERROR: column \"test_structs.string_\" must appear in the GROUP BY clause or be used in an aggregate function (SQLSTATE 42803)",
		},
		{
			desc: "select derived & non-primary field wo/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString),
				).ProtoQuery(),
			resultStruct: DerivedStruct4{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1",
			expectedResult: []*DerivedStruct4{},
			expectedError:  "ERROR: column \"test_structs_nesteds.nested\" must appear in the GROUP BY clause or be used in an aggregate function (SQLSTATE 42803)",
		},
		{
			desc: "select derived & non-primary field w/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString),
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct5{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct5{
				{1, "nested_acs"},
				{2, "nested_bcs_1"},
				{1, "nested_bcs_2"},
			},
		},
		{
			desc: "select derived & primary field w/ group by non-primary field",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestString),
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct6{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"jsonb_agg(test_structs.String_) as test_string, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct6{
				{1, []string{"acs"}, "nested_acs"},
				{2, []string{"bcs", "bcs"}, "nested_bcs_1"},
				{1, []string{"bcs"}, "nested_bcs_2"},
			},
		},
		{
			desc: "select derived w/ filter",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestKey).
						AggrFunc(aggregatefunc.Count).
						Filter(
							"test_string_affected_by_enum1",
							search.NewQueryBuilder().
								AddExactMatches(search.TestEnum, storage.TestStruct_ENUM1.String()).ProtoQuery(),
						),
					search.NewQuerySelect(search.TestKey).
						AggrFunc(aggregatefunc.Count).
						Filter(
							"test_string_affected_by_enum2",
							search.NewQueryBuilder().
								AddExactMatches(search.TestEnum, storage.TestStruct_ENUM2.String()).ProtoQuery(),
						),
				).ProtoQuery(),
			resultStruct: DerivedStruct7{},
			expectedQuery: "select count(test_structs.Key1) filter (where (test_structs.Enum = $1)) as test_string_affected_by_enum1, " +
				"count(test_structs.Key1) filter (where (test_structs.Enum = $2)) as test_string_affected_by_enum2 " +
				"from test_structs",
			expectedResult: []*DerivedStruct7{
				{2, 2},
			},
		},
		{
			desc: "select derived w/ filter and group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestKey).
						AggrFunc(aggregatefunc.Count).
						Filter(
							"test_string_affected_by_enum1",
							search.NewQueryBuilder().
								AddExactMatches(search.TestEnum, storage.TestStruct_ENUM1.String()).ProtoQuery(),
						),
					search.NewQuerySelect(search.TestKey).
						AggrFunc(aggregatefunc.Count).
						Filter(
							"test_string_affected_by_enum2",
							search.NewQueryBuilder().
								AddExactMatches(search.TestEnum, storage.TestStruct_ENUM2.String()).ProtoQuery(),
						),
				).AddGroupBy(search.TestBool).ProtoQuery(),
			resultStruct: DerivedStruct8{},
			expectedQuery: "select count(test_structs.Key1) filter (where (test_structs.Enum = $1)) as test_string_affected_by_enum1, " +
				"count(test_structs.Key1) filter (where (test_structs.Enum = $2)) as test_string_affected_by_enum2, " +
				"test_structs.Bool as test_bool from test_structs " +
				"group by test_structs.Bool",
			expectedResult: []*DerivedStruct8{
				{1, 1, false},
				{1, 1, true},
			},
		},
		{
			desc: "select multiple derived w/ group by & pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count),
				).
				AddGroupBy(search.TestNestedString).
				WithPagination(
					search.NewPagination().AddSortOption(
						search.NewSortOption(search.TestNestedString).Reversed(true),
					),
				).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"count(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested order by test_structs_nesteds.Nested desc",
			expectedResult: []*DerivedStruct3{
				{1, 1, "nested_bcs_2"},
				{2, 2, "nested_bcs_1"},
				{1, 1, "nested_acs"},
			},
		},
		{
			desc: "select multiple derived w/ group by & derived field pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestNestedString2).AggrFunc(aggregatefunc.Count),
				).
				AddGroupBy(search.TestNestedString).WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.TestNestedString).
						AggregateBy(aggregatefunc.Count, false).
						Reversed(true),
				).Limit(1),
			).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"count(test_structs_nesteds.Nested2_Nested2) as test_nested_string_2_count, " +
				"test_structs_nesteds.Nested as test_nested_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested order by count(test_structs_nesteds.Nested) desc LIMIT 1",
			expectedResult: []*DerivedStruct3{
				{2, 2, "nested_bcs_1"},
			},
		},
		{
			desc: "select derived & primary field w/ group by non-primary field & pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
				).
				AddGroupBy(search.TestNestedString).WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.TestString),
				),
			).ProtoQuery(),
			resultStruct: DerivedStruct6{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"test_structs_nesteds.Nested as test_nested_string, " +
				"jsonb_agg(test_structs.String_) as test_string " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested order by test_structs.String_ asc",
			expectedError: "column \"test_structs.string_\" must appear in the GROUP BY clause or be used in an aggregate function",
		},
		{
			desc: "select derived & primary field w/ group by non-primary field & pagination",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
					search.NewQuerySelect(search.TestString),
				).
				AddGroupBy(search.TestNestedString).WithPagination(
				search.NewPagination().
					AddSortOption(
						search.NewSortOption(search.TestString).
							AggregateBy(aggregatefunc.Count, false).Reversed(true),
					).AddSortOption(
					search.NewSortOption(search.TestNestedString),
				),
			).ProtoQuery(),
			resultStruct: DerivedStruct9{},
			expectedQuery: "select count(test_structs_nesteds.Nested) as test_nested_string_count, " +
				"jsonb_agg(test_structs.String_) as test_string, " +
				"test_structs_nesteds.Nested as test_nested_string, " +
				"count(test_structs.String_) as test_string_count " +
				"from test_structs inner join test_structs_nesteds " +
				"on test_structs.Key1 = test_structs_nesteds.test_structs_Key1 " +
				"group by test_structs_nesteds.Nested " +
				"order by count(test_structs.String_) desc, test_structs_nesteds.Nested asc",
			expectedResult: []*DerivedStruct9{
				{2, []string{"bcs", "bcs"}, 2, "nested_bcs_1"},
				{1, []string{"acs"}, 1, "nested_acs"},
				{1, []string{"bcs"}, 1, "nested_bcs_2"},
			},
		},
		{
			desc: "nil query",
			q:    nil,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			pgSearch.AssertSQLQueryString(t, c.q, schema.TestStructsSchema, c.expectedQuery)
			runTest(ctx, t, testDB, c)
		})
	}
}

func getTestStructs() []*storage.TestStruct {
	return []*storage.TestStruct{
		{
			Key1:    uuid.NewV4().String(),
			String_: "acs",
			Bool:    true,
			Enum:    storage.TestStruct_ENUM1,
			Nested: []*storage.TestStruct_Nested{
				{
					Nested:   "nested_acs",
					IsNested: false,
					Nested2: &storage.TestStruct_Nested_Nested2{
						Nested2: "nested_acs_nested_1",
					},
				},
			},
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "acs",
			Bool:    false,
			Enum:    storage.TestStruct_ENUM1,
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "bcs",
			Bool:    true,
			Enum:    storage.TestStruct_ENUM2,
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "bcs",
			Bool:    false,
			Enum:    storage.TestStruct_ENUM2,
			Nested: []*storage.TestStruct_Nested{
				{
					Nested: "nested_bcs_1",
				},
				{
					Nested: "nested_bcs_1",
					Nested2: &storage.TestStruct_Nested_Nested2{
						Nested2: "nested_bcs_nested_1",
					},
				},
				{
					Nested: "nested_bcs_2",
					Nested2: &storage.TestStruct_Nested_Nested2{
						Nested2: "nested_bcs_nested_2",
					},
				},
			},
		},
	}
}

func runTest(ctx context.Context, t *testing.T, testDB *pgtest.TestPostgres, tc selectQTestCase) {
	var results any
	var err error
	switch tc.resultStruct.(type) {
	case Struct1:
		results, err = pgSearch.RunSelectRequestForSchema[Struct1](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2GrpBy1:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2GrpBy1](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2GrpBy2:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2GrpBy2](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2GrpBy3:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2GrpBy3](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2GrpBy4:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2GrpBy4](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct2GrpBy5:
		results, err = pgSearch.RunSelectRequestForSchema[Struct2GrpBy5](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct3:
		results, err = pgSearch.RunSelectRequestForSchema[Struct3](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct4:
		results, err = pgSearch.RunSelectRequestForSchema[Struct4](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case Struct5:
		results, err = pgSearch.RunSelectRequestForSchema[Struct5](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct1:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct1](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct2:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct2](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct22:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct22](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct3:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct3](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct4:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct4](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct5:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct5](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct6:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct6](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct7:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct7](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct8:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct8](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	case DerivedStruct9:
		results, err = pgSearch.RunSelectRequestForSchema[DerivedStruct9](ctx, testDB.DB, schema.TestStructsSchema, tc.q)
	}
	if tc.expectedError != "" {
		assert.Error(t, err, tc.expectedError)
		return
	}
	assert.NoError(t, err)

	if tc.q == nil {
		assert.Nil(t, results)
		return
	}

	if tc.q.GetPagination() != nil {
		assert.Equal(t, tc.expectedResult, results)
	} else {
		assert.ElementsMatch(t, tc.expectedResult, results)
	}
}
