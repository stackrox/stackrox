//go:build sql_integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/sync"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContexCancellationInWalk(t *testing.T) {

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	store := postgres.New(testDB.DB)

	testStructs := getTestStructs()

	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	ctxWithCancel, cancel := context.WithCancelCause(ctx)

	count := 0
	err := store.Walk(ctxWithCancel, func(obj *storage.TestStruct) error {
		cancel(errors.New("cancelling context on the first read"))
		count++
		return nil
	})
	assert.Equal(t, 1, count)
	assert.ErrorIs(t, err, context.Canceled)
}

func queryCursorFromStatActivity(t *testing.T, hint string) string {
	t.Helper()
	ctx := context.Background()
	adminConn, err := pgx.Connect(ctx, conn.GetConnectionStringWithDatabaseName(t, "postgres"))
	require.NoError(t, err)
	defer func() { _ = adminConn.Close(ctx) }()

	var query string
	err = adminConn.QueryRow(ctx,
		"SELECT query FROM pg_stat_activity WHERE query LIKE '%FETCH%' AND query LIKE $1 AND state = 'idle in transaction'",
		"%"+hint+"%",
	).Scan(&query)
	if err != nil {
		return ""
	}
	return query
}

func TestCursorNameContainsHint(t *testing.T) {
	testDB := pgtest.ForT(t)
	ctx := sac.WithAllAccess(context.Background())
	store := postgres.New(testDB.DB)

	for _, s := range getTestStructs() {
		require.NoError(t, store.Upsert(ctx, s))
	}

	cursorReady := make(chan struct{}, 1)
	queryDone := make(chan struct{})

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = store.Walk(ctx, func(_ *storage.TestStruct) error {
			select {
			case cursorReady <- struct{}{}:
			default:
			}
			<-queryDone
			return nil
		})
	})

	<-cursorReady

	foundQuery := queryCursorFromStatActivity(t, "Walk")
	close(queryDone)
	wg.Wait()

	require.NotEmpty(t, foundQuery, "expected to find a FETCH statement with the cursor hint in pg_stat_activity")

	parts := strings.Fields(foundQuery)
	require.GreaterOrEqual(t, len(parts), 4, "expected FETCH N FROM cursor_name")
	cursorName := parts[len(parts)-1]

	assert.True(t, strings.HasPrefix(cursorName, "test_structs_Walk_"),
		"cursor name %q should start with test_structs_Walk_", cursorName)

	t.Logf("cursor name: %s", cursorName)
}

func TestCursorNameWithWalkByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	ctx := sac.WithAllAccess(context.Background())

	store := postgres.New(testDB.DB)

	testStructs := getTestStructs()
	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	cursorReady := make(chan struct{}, 1)
	queryDone := make(chan struct{})

	var wg sync.WaitGroup
	wg.Go(func() {
		q := search.NewQueryBuilder().AddExactMatches(search.TestKey, testStructs[0].GetKey1()).ProtoQuery()
		_ = store.WalkByQuery(ctx, q, func(_ *storage.TestStruct) error {
			select {
			case cursorReady <- struct{}{}:
			default:
			}
			<-queryDone
			return nil
		})
	})

	<-cursorReady

	foundQuery := queryCursorFromStatActivity(t, "WalkByQuery")
	close(queryDone)
	wg.Wait()

	require.NotEmpty(t, foundQuery, "expected to find a FETCH statement with WalkByQuery hint in pg_stat_activity")

	parts := strings.Fields(foundQuery)
	require.GreaterOrEqual(t, len(parts), 4)
	cursorName := parts[len(parts)-1]

	assert.True(t, strings.HasPrefix(cursorName, "test_structs_WalkByQuery_"),
		"cursor name %q should start with test_structs_WalkByQuery_", cursorName)

	t.Logf("cursor name: %s", cursorName)
}

func TestRunDistinctCountForSchema(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	store := postgres.New(testDB.DB)
	testStructs := getTestStructs()
	for _, s := range testStructs {
		require.NoError(t, store.Upsert(ctx, s))
	}

	// 4 structs with unique keys
	count, err := pgSearch.RunDistinctCountForSchema(ctx, testDB.DB, schema.TestStructsSchema, search.EmptyQuery(), search.TestKey)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// 4 structs with 2 distinct string values ("acs", "bcs")
	count, err = pgSearch.RunDistinctCountForSchema(ctx, testDB.DB, schema.TestStructsSchema, search.EmptyQuery(), search.TestString)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Filtered: only "acs" strings, 2 distinct keys
	q := search.NewQueryBuilder().AddExactMatches(search.TestString, "acs").ProtoQuery()
	count, err = pgSearch.RunDistinctCountForSchema(ctx, testDB.DB, schema.TestStructsSchema, q, search.TestKey)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Empty result set
	q = search.NewQueryBuilder().AddExactMatches(search.TestString, "nonexistent").ProtoQuery()
	count, err = pgSearch.RunDistinctCountForSchema(ctx, testDB.DB, schema.TestStructsSchema, q, search.TestKey)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestLargeParameterSetUsesANY(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	store := postgres.New(testDB.DB)

	// Insert 10 test structs with known keys.
	knownKeys := make([]string, 10)
	for i := range knownKeys {
		key := uuid.NewV4().String()
		knownKeys[i] = key
		require.NoError(t, store.Upsert(ctx, &storage.TestStruct{
			Key1:    key,
			Key2:    fmt.Sprintf("key2-%d", i),
			String_: fmt.Sprintf("value-%d", i),
		}))
	}

	// Build a query with 70K+ values for the same field.
	// The first 10 are the known keys, the rest are random UUIDs that won't match.
	values := make([]string, 70_001)
	copy(values, knownKeys)
	for i := 10; i < len(values); i++ {
		values[i] = uuid.NewV4().String()
	}

	// This would fail with "too many parameters" if IN were used (65535 limit).
	// With ANY it uses a single array parameter.
	q := search.NewQueryBuilder().AddExactMatches(search.TestKey, values...).ProtoQuery()
	results, err := store.Search(ctx, q)
	require.NoError(t, err)
	assert.Len(t, results, 10, "should find all 10 inserted structs")
}
