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

func TestSelectQueryResults(t *testing.T) {
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
				AddSelectFields(search.TestString).ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_multi_key_structs.String_ teststring from test_multi_key_structs",
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
				AddSelectFields(search.TestString).
				AddExactMatches(search.TestString, "acs").ProtoQuery(),
			resultStruct:  Struct1{},
			expectedQuery: "select test_multi_key_structs.String_ teststring from test_multi_key_structs where test_multi_key_structs.String_ = $1",
			expectedResult: []*Struct1{
				{"acs"},
				{"acs"},
			},
		},
		{
			desc: "child schema; multiple select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.TestNestedString, search.TestNestedBool).
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct2{},
			expectedQuery: "select test_multi_key_structs_nesteds.Nested testnestedstring, test_multi_key_structs_nesteds.IsNested testnestedbool " +
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
				AddSelectFields(search.TestNestedString, search.TestNestedBool).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) testnestedbool, test_multi_key_structs.String_ teststring " +
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
				AddSelectFields(search.TestNestedString, search.TestNestedBool).
				AddGroupBy(search.TestString).ProtoQuery(),
			resultStruct: Struct2GrpBy1{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) testnestedbool, test_multi_key_structs.String_ teststring " +
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
					TestNestedString: []string{"nested_bcs"},
					TestNestedBool:   []bool{false},
					TestString:       "bcs",
				},
			},
		},
		{
			desc: "child schema; multiple select w/ where & multiple group by",
			q: search.NewQueryBuilder().
				AddSelectFields(search.TestNestedString, search.TestNestedBool).
				AddExactMatches(search.TestNestedString, "nested_acs").
				AddGroupBy(search.TestString, search.TestBool).ProtoQuery(),
			resultStruct: Struct2GrpBy2{},
			expectedQuery: "select jsonb_agg(test_multi_key_structs_nesteds.Nested) testnestedstring, jsonb_agg(test_multi_key_structs_nesteds.IsNested) testnestedbool, " +
				"test_multi_key_structs.String_ teststring, test_multi_key_structs.Bool testbool " +
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
				AddSelectFields(search.TestString, search.TestNestedString).ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_multi_key_structs.String_ teststring, test_multi_key_structs_nesteds.Nested testnestedstring " +
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
					TestNestedString: "nested_bcs",
				},
			},
		},
		{
			desc: "base schema and child schema conjunction query; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.TestString, search.TestNestedString).
				AddExactMatches(search.TestString, "acs").
				AddExactMatches(search.TestNestedString, "nested_acs").ProtoQuery(),
			resultStruct: Struct3{},
			expectedQuery: "select test_multi_key_structs.String_ teststring, test_multi_key_structs_nesteds.Nested testnestedstring " +
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
					Nested: "nested_bcs",
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
	}
	return results, err
}
