//go:build sql_integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunSelectCursorForSchemaFn verifies that RunSelectCursorForSchemaFn
// produces the same results as RunSelectRequestForSchemaFn. The cursor
// path uses DECLARE CURSOR / FETCH N under the hood, so equivalence with
// the non-cursor path is the key property.
func TestRunSelectCursorForSchemaFn(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	store := postgres.New(testDB.DB)
	for _, s := range getTestStructs() {
		require.NoError(t, store.Upsert(ctx, s))
	}

	t.Run("select single field matches non-cursor path", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			ProtoQuery()

		var expected []*Struct1
		err := pgSearch.RunSelectRequestForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)
		require.NotEmpty(t, expected)

		var actual []*Struct1
		err = pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("select with where clause", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			AddExactMatches(search.TestString, "acs").
			ProtoQuery()

		var expected []*Struct1
		err := pgSearch.RunSelectRequestForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)

		var actual []*Struct1
		err = pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("select multiple fields from child table", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(
				search.NewQuerySelect(search.TestNestedString),
				search.NewQuerySelect(search.TestNestedBool),
			).
			AddExactMatches(search.TestNestedString, "nested_acs").
			ProtoQuery()

		var expected []*Struct2
		err := pgSearch.RunSelectRequestForSchemaFn[Struct2](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct2) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)

		var actual []*Struct2
		err = pgSearch.RunSelectCursorForSchemaFn[Struct2](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct2) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("select with pagination preserves order", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(
				search.NewQuerySelect(search.TestString),
				search.NewQuerySelect(search.TestNestedString),
			).
			WithPagination(
				search.NewPagination().
					AddSortOption(search.NewSortOption(search.TestString)).
					AddSortOption(search.NewSortOption(search.TestNestedString)),
			).
			ProtoQuery()

		var expected []*Struct3
		err := pgSearch.RunSelectRequestForSchemaFn[Struct3](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct3) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)

		var actual []*Struct3
		err = pgSearch.RunSelectCursorForSchemaFn[Struct3](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct3) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("select with derived fields and group by", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(
				search.NewQuerySelect(search.TestNestedString).AggrFunc(aggregatefunc.Count),
			).
			AddGroupBy(search.TestNestedString).
			ProtoQuery()

		var expected []*DerivedStruct5
		err := pgSearch.RunSelectRequestForSchemaFn[DerivedStruct5](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *DerivedStruct5) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)

		var actual []*DerivedStruct5
		err = pgSearch.RunSelectCursorForSchemaFn[DerivedStruct5](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *DerivedStruct5) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("nil query returns no results", func(t *testing.T) {
		called := false
		err := pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, nil, func(r *Struct1) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("empty result set", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			AddExactMatches(search.TestString, "nonexistent_value_xyz").
			ProtoQuery()

		var actual []*Struct1
		err := pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("callback error propagates and stops iteration", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			ProtoQuery()

		expectedErr := errors.New("callback failed")
		callCount := 0
		err := pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			callCount++
			return expectedErr
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, callCount)
	})

	t.Run("callback error after multiple rows", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			ProtoQuery()

		expectedErr := errors.New("fail on third")
		callCount := 0
		err := pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			callCount++
			if callCount == 3 {
				return expectedErr
			}
			return nil
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 3, callCount)
	})

	t.Run("select null timestamp field", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(
				search.NewQuerySelect(search.TestString),
				search.NewQuerySelect(search.TestTimestamp),
			).
			ProtoQuery()

		var expected []*Struct5
		err := pgSearch.RunSelectRequestForSchemaFn[Struct5](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct5) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)

		var actual []*Struct5
		err = pgSearch.RunSelectCursorForSchemaFn[Struct5](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct5) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("select with limit", func(t *testing.T) {
		q := search.NewQueryBuilder().
			AddSelectFields(search.NewQuerySelect(search.TestString)).
			WithPagination(
				search.NewPagination().
					AddSortOption(search.NewSortOption(search.TestString)).
					Limit(2),
			).
			ProtoQuery()

		var expected []*Struct1
		err := pgSearch.RunSelectRequestForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			expected = append(expected, r)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, expected, 2)

		var actual []*Struct1
		err = pgSearch.RunSelectCursorForSchemaFn[Struct1](ctx, testDB.DB, schema.TestStructsSchema, q, func(r *Struct1) error {
			actual = append(actual, r)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
