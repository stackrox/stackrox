//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubQueryComposition(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(t)
	store := newStore(db)
	for _, obj := range []*storage.TestSingleKeyStruct{
		{Key: "first", Name: "one", Int64: 1},
		{Key: "second", Name: "two", Int64: 2},
		{Key: "third", Name: "three", Int64: 3},
		{Key: "null-name"},
	} {
		require.NoError(t, store.Upsert(ctx, obj))
	}
	_, err := db.Exec(ctx, "UPDATE test_single_key_structs SET name = NULL WHERE key = $1", "null-name")
	require.NoError(t, err)
	category := v1.SearchCategory(100)
	inner := search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.TestKey)).
		AddStrings(search.TestInt64, ">1").ProtoQuery()
	membership := search.MatchFieldSubQuery(search.TestKey, category, inner)
	nestedQuery := search.ConjunctionQuery(membership, search.NewQueryBuilder().AddStrings(search.TestInt64, ">2").ProtoQuery())
	nestedQuery.Selects = inner.GetSelects()
	nested := search.MatchFieldSubQuery(search.TestKey, category, nestedQuery)
	nullMembership := search.MatchFieldSubQuery(search.TestName, category,
		search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.TestName)).
			AddExactMatches(search.TestKey, "null-name").ProtoQuery())
	distinct := inner.CloneVT()
	distinct.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.TestKey).Distinct().Proto()}
	distinct.Pagination = search.NewPagination().AddSortOption(search.NewSortOption(search.TestInt64)).Proto()
	allRows := search.ConjunctionQuery(search.EmptyQuery(), search.EmptyQuery())
	allRows.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.TestKey).Proto()}
	limited := inner.CloneVT()
	limited.Pagination = search.NewPagination().Limit(1).Offset(1).AddSortOption(search.NewSortOption(search.TestKey)).Proto()

	for name, test := range map[string]struct {
		query *v1.Query
		keys  []string
	}{
		"same category": {query: membership, keys: []string{"second", "third"}},
		"conjunction": {query: search.ConjunctionQuery(membership,
			search.NewQueryBuilder().AddStrings(search.TestInt64, "<3").ProtoQuery()), keys: []string{"second"}},
		"disjunction": {query: search.DisjunctionQuery(membership,
			search.NewQueryBuilder().AddExactMatches(search.TestKey, "first").ProtoQuery()), keys: []string{"first", "second", "third"}},
		"negation": {query: search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{membership}}),
			keys: []string{"first", "null-name"}},
		"nested":                 {query: nested, keys: []string{"third"}},
		"empty inner predicates": {query: search.MatchFieldSubQuery(search.TestKey, category, allRows), keys: []string{"first", "second", "third", "null-name"}},
		"ordered pagination":     {query: search.MatchFieldSubQuery(search.TestKey, category, limited), keys: []string{"third"}},
		"distinct with ordering": {query: search.MatchFieldSubQuery(search.TestKey, category, distinct), keys: []string{"second", "third"}},
		"unsupported inner filter": {query: search.MatchFieldSubQuery(search.TestKey, category,
			search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.TestKey)).
				AddExactMatches(search.DeploymentName, "unsupported").ProtoQuery())},
		"null selected value": {query: nullMembership},
		"negated null membership": {query: search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{nullMembership}}),
			keys: []string{"first", "second", "third", "null-name"}},
	} {
		t.Run(name, func(t *testing.T) {
			count, err := store.Count(ctx, test.query.CloneVT())
			require.NoError(t, err)
			assert.Equal(t, len(test.keys), count)
			rows, err := store.GetByQuery(ctx, test.query.CloneVT())
			require.NoError(t, err)
			var keys []string
			for _, row := range rows {
				keys = append(keys, row.GetKey())
			}
			assert.ElementsMatch(t, test.keys, keys)
			results, err := store.Search(ctx, test.query.CloneVT())
			require.NoError(t, err)
			keys = nil
			for _, result := range results {
				keys = append(keys, result.ID)
			}
			assert.ElementsMatch(t, test.keys, keys)

			query := test.query.CloneVT()
			query.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.TestKey).Proto()}
			keys = nil
			err = RunSelectRequestForSchemaFn(ctx, db, schema.TestSingleKeyStructsSchema, query, func(row *struct {
				Key string `db:"test_key"`
			}) error {
				keys = append(keys, row.Key)
				return nil
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, test.keys, keys)
			keys = nil
			err = store.WalkByQuery(ctx, test.query.CloneVT(), func(row *storage.TestSingleKeyStruct) error {
				keys = append(keys, row.GetKey())
				return nil
			})
			require.NoError(t, err)
			assert.ElementsMatch(t, test.keys, keys)
		})
	}
}
