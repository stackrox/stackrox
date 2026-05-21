//go:build sql_integration

package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/sync"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	conn "github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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
