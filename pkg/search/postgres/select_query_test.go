//go:build sql_integration

package postgres_test

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pkgPG "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Struct1 struct {
	TestString string `db:"teststring"`
}

type Struct2 struct {
	TestNestedString string `db:"testnestedstring"`
	TestNestedBool   bool   `db:"testnestedbool"`
}

type Struct2GrpBy1 struct {
	TestNestedString []string `db:"testnestedstring"`
	TestNestedBool   []bool   `db:"testnestedbool"`
	TestString       string   `db:"teststring"`
}

type Struct2GrpBy2 struct {
	TestNestedString []string `db:"testnestedstring"`
	TestNestedBool   []bool   `db:"testnestedbool"`
	TestString       string   `db:"teststring"`
	TestBool         bool     `db:"testbool"`
}

type Struct3 struct {
	TestString       string `db:"teststring"`
	TestNestedString string `db:"testnestedstring"`
}

func TestSelectQuery(t *testing.T) {
	t.Parallel()

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	defer testDB.Teardown(t)

	store := postgres.New(testDB.Pool)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	for _, c := range []struct {
		desc           string
		q              *v1.Query
		resultStruct   any
		expectedError  string
		expectedQuery  string
		expectedResult any
	}{
		{
			desc: "base schema; no select",
			q: search.NewQueryBuilder().
				AddExactMatches(search.TestString, "acs").ProtoQuery(),
			resultStruct:   Struct1{},
			expectedError:  "select portion of the query cannot be empty",
			expectedResult: nil,
		},
		{
			desc: "base schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field: search.TestString.String(),
					},
				).ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_multi_key_structs.String_ as teststring from test_multi_key_structs",
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
				AddSelectFields(&v1.QuerySelect{
					Field: search.TestString.String(),
				},
				).
				AddExactMatches(search.TestString, "acs").ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_multi_key_structs.String_ as teststring from test_multi_key_structs where test_multi_key_structs.String_ = $1",
			expectedResult: []*Struct1{
				{"acs"},
				{"acs"},
			},
		},
		{
			desc: "child schema; multiple select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedBool.String(),
					},
				).
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct2{},
			expectedQuery: "select test_multi_key_structs_nesteds.Nested as testnestedstring, test_multi_key_structs_nesteds.IsNested as testnestedbool " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs_nesteds.Nested = $",
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
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedBool.String(),
					},
				).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) as testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) as testnestedbool, test_multi_key_structs.String_ as teststring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs_nesteds.Nested = $1 " +
				"group by test_multi_key_structs.String_",
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
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedBool.String(),
					},
				).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) as testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) as testnestedbool, test_multi_key_structs.String_ as teststring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"group by test_multi_key_structs.String_",
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
			desc: "child schema; multiple select w/ where & multiple group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedBool.String(),
					},
				).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString, search.TestBool).ProtoQuery(),
			resultStruct: Struct2GrpBy2{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) as testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) as testnestedbool, " +
				"test_multi_key_structs.String_ as teststring, test_multi_key_structs.Bool as testbool " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs_nesteds.Nested = $1 " +
				"group by test_multi_key_structs.String_, test_multi_key_structs.Bool",
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
					&v1.QuerySelect{
						Field: search.TestString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
				).ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_multi_key_structs.String_ as teststring, test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2",
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
					&v1.QuerySelect{
						Field: search.TestString.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
				).
				AddExactMatches(search.TestString, "acs").
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_multi_key_structs.String_ as teststring, test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where (test_multi_key_structs_nesteds.Nested = $1 and test_multi_key_structs.String_ = $2)",
			expectedResult: []*Struct3{
				{
					TestString:       "acs",
					TestNestedString: "nested_acs",
				},
			},
		},
		{
			desc: "nil query",
			q:    nil,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			results, err := runTests(ctx, testDB, c.q, c.resultStruct)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
				return
			}
			assert.NoError(t, err)

			if c.q == nil {
				assert.Nil(t, results)
				return
			}

			assert.EqualValues(t, c.expectedResult, results)
		})
	}
}

type DerivedStruct1 struct {
	TestNestedStringCount int `db:"testnestedstringcount"`
}

type DerivedStruct2 struct {
	TestNestedStringCount  int `db:"testnestedstringcount"`
	TestNestedString2Count int `db:"testnestedstring2count"`
}

type DerivedStruct22 struct {
	TestNestedStringCount int    `db:"testnestedstringcount"`
	TopTestNestedString2  string `db:"testnestedstring2max"`
}

type DerivedStruct3 struct {
	TestNestedStringCount  int    `db:"testnestedstringcount"`
	TestNestedString2Count int    `db:"testnestedstring2count"`
	TestNestedString       string `db:"testnestedstring"`
}

type DerivedStruct4 struct {
	TestNestedStringCount int    `db:"testnestedstringcount"`
	TestString            string `db:"teststring"`
}

type DerivedStruct5 struct {
	TestNestedStringCount int    `db:"testnestedstringcount"`
	TestNestedString      string `db:"testnestedstring"`
}

type DerivedStruct6 struct {
	TestNestedStringCount int      `db:"testnestedstringcount"`
	TestString            []string `db:"teststring"`
	TestNestedString      string   `db:"testnestedstring"`
}

func TestSelectDerivedFieldQuery(t *testing.T) {
	t.Parallel()

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	defer testDB.Teardown(t)

	store := postgres.New(testDB.Pool)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	for _, c := range []struct {
		desc           string
		q              *v1.Query
		resultStruct   any
		expectedError  string
		expectedQuery  string
		expectedResult any
	}{
		{
			desc: "select one derived",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 ",
			expectedResult: []*DerivedStruct1{
				{4},
			},
		},
		{
			desc: "select one derived w/ distinct",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
						Distinct:      true,
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 ",
			expectedResult: []*DerivedStruct1{
				{3},
			},
		},
		{
			desc: "select multiple derived",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
						Distinct:      true,
					},
					&v1.QuerySelect{
						Field:         search.TestNestedString2.String(),
						AggregateFunc: pkgPG.Count.String(),
						Distinct:      true,
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct2{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"count(test_multi_key_structs_nesteds.Nested2_Nested2) as testnestedstring2count " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2",
			expectedResult: []*DerivedStruct2{
				{3, 4},
			},
		},
		{
			desc: "select multiple derived again",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field:         search.TestNestedString2.String(),
						AggregateFunc: pkgPG.Max.String(),
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct22{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"max(test_multi_key_structs_nesteds.Nested2_Nested2) as testnestedstring2max " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2",
			expectedResult: []*DerivedStruct22{
				{4, "nested_bcs_nested_2"},
			},
		},
		{
			desc: "select multiple derived w/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field:         search.TestNestedString2.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"count(test_multi_key_structs_nesteds.Nested2_Nested2) as testnestedstring2count, test_multi_key_structs.String_ as teststring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"group by test_multi_key_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct3{
				{1, 1, "nested_acs"},
				{2, 2, "nested_bcs_1"},
				{1, 1, "nested_bcs_2"},
			},
		},
		{
			desc: "select one derived w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
				).
				AddExactMatches(search.TestString, "bcs").ProtoQuery(),
			resultStruct: DerivedStruct1{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs.String_ = $1 ",
			expectedResult: []*DerivedStruct1{
				{3},
			},
		},
		{
			desc: "select multiple derived w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field:         search.TestNestedString2.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
				).
				AddStrings(search.TestNestedString2, "nested").ProtoQuery(),
			resultStruct: DerivedStruct2{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"count(test_multi_key_structs_nesteds.Nested2_Nested2) as testnestedstring2count " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs_nesteds.Nested2_Nested2 ilike $1 ",
			expectedResult: []*DerivedStruct2{
				{3, 3},
			},
		},
		{
			desc: "select multiple derived w/ where & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field:         search.TestNestedString2.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
				).
				AddStrings(search.TestNestedString2, "nested").
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct3{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"count(test_multi_key_structs_nesteds.Nested2_Nested2) as testnestedstring2count, test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"where test_multi_key_structs_nesteds.Nested2_Nested2 ilike $1 " +
				"group by test_multi_key_structs_nesteds.Nested",
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
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field: search.TestString.String(),
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct4{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, test_multi_key_structs.String_ as teststring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 ",
			expectedResult: []*DerivedStruct4{
				{1, "acs"},
				{3, "bcs"},
			},
			expectedError: "ERROR: column \"test_multi_key_structs.string_\" must appear in the GROUP BY clause or be used in an aggregate function (SQLSTATE 42803)",
		},
		{
			desc: "select derived & non-primary field wo/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
				).ProtoQuery(),
			resultStruct: DerivedStruct4{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, " +
				"test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"group by test_multi_key_structs.Key1, test_multi_key_structs.Key",
			expectedResult: []*DerivedStruct4{},
			expectedError:  "ERROR: column \"test_multi_key_structs_nesteds.nested\" must appear in the GROUP BY clause or be used in an aggregate function (SQLSTATE 42803)",
		},
		{
			desc: "select derived & non-primary field w/ group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field: search.TestNestedString.String(),
					},
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct5{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"group by test_multi_key_structs_nesteds.Nested",
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
					&v1.QuerySelect{
						Field:         search.TestNestedString.String(),
						AggregateFunc: pkgPG.Count.String(),
					},
					&v1.QuerySelect{
						Field: search.TestString.String(),
					},
				).
				AddGroupBy(search.TestNestedString).ProtoQuery(),
			resultStruct: DerivedStruct6{},
			expectedQuery: "select count(test_multi_key_structs_nesteds.Nested) as testnestedstringcount, test_multi_key_structs_nesteds.Nested as testnestedstring " +
				"from test_multi_key_structs inner join test_multi_key_structs_nesteds " +
				"on test_multi_key_structs.Key1 = test_multi_key_structs_nesteds.test_multi_key_structs_Key1 " +
				"and test_multi_key_structs.Key2 = test_multi_key_structs_nesteds.test_multi_key_structs_Key2 " +
				"group by test_multi_key_structs_nesteds.Nested",
			expectedResult: []*DerivedStruct6{
				{1, []string{"acs"}, "nested_acs"},
				{2, []string{"bcs", "bcs"}, "nested_bcs_1"},
				{1, []string{"bcs"}, "nested_bcs_2"},
			},
		},
		{
			desc: "nil query",
			q:    nil,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			results, err := runTests(ctx, testDB, c.q, c.resultStruct)
			if c.expectedError != "" {
				assert.ErrorContains(t, err, c.expectedError)
				return
			}
			assert.NoError(t, err)

			if c.q == nil {
				assert.Nil(t, results)
				return
			}

			assert.ElementsMatch(t, c.expectedResult, results)
		})
	}
}

func getTestStructs() []*storage.TestMultiKeyStruct {
	return []*storage.TestMultiKeyStruct{
		{
			Key1:    uuid.NewV4().String(),
			String_: "acs",
			Bool:    true,
			Enum:    storage.TestMultiKeyStruct_ENUM1,
			Nested: []*storage.TestMultiKeyStruct_Nested{
				{
					Nested:   "nested_acs",
					IsNested: false,
					Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
						Nested2: "nested_acs_nested_1",
					},
				},
			},
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "acs",
			Bool:    false,
			Enum:    storage.TestMultiKeyStruct_ENUM1,
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "bcs",
			Bool:    true,
			Enum:    storage.TestMultiKeyStruct_ENUM2,
		},
		{
			Key1:    uuid.NewV4().String(),
			String_: "bcs",
			Bool:    false,
			Enum:    storage.TestMultiKeyStruct_ENUM2,
			Nested: []*storage.TestMultiKeyStruct_Nested{
				{
					Nested: "nested_bcs_1",
				},
				{
					Nested: "nested_bcs_1",
					Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
						Nested2: "nested_bcs_nested_1",
					},
				},
				{
					Nested: "nested_bcs_2",
					Nested2: &storage.TestMultiKeyStruct_Nested_Nested2{
						Nested2: "nested_bcs_nested_2",
					},
				},
			},
		},
	}
}

func runTests(ctx context.Context, testDB *pgtest.TestPostgres, q *v1.Query, resultStruct any) (any, error) {
	var results any
	var err error
	switch resultStruct.(type) {
	case Struct1:
		results, err = pkgPG.RunSelectRequestForSchema[Struct1](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case Struct2:
		results, err = pkgPG.RunSelectRequestForSchema[Struct2](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case Struct2GrpBy1:
		results, err = pkgPG.RunSelectRequestForSchema[Struct2GrpBy1](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case Struct2GrpBy2:
		results, err = pkgPG.RunSelectRequestForSchema[Struct2GrpBy2](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case Struct3:
		results, err = pkgPG.RunSelectRequestForSchema[Struct3](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct1:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct1](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct2:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct2](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct22:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct22](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct3:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct3](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct4:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct4](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct5:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct5](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	case DerivedStruct6:
		results, err = pkgPG.RunSelectRequestForSchema[DerivedStruct6](ctx, testDB.Pool, schema.TestMultiKeyStructsSchema, q)
	}
	return results, err
}
