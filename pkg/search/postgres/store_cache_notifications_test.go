//go:build sql_integration

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/require"
)

type receivedCacheNotification struct{ channel, payload string }

func TestCacheTransactionalTriggers(t *testing.T) {
	for name, keys := range map[string]struct{ keyType, first, second string }{
		"text and quoted identifiers": {"text", `first'\\key`, `second"key`},
		"UUID primary key":            {"uuid", "12345678-0000-4000-8000-000000000001", "12345678-0000-4000-8000-000000000002"},
		"oversized key":               {"text", strings.Repeat(`\\"`, 4000), "small"},
	} {
		t.Run(name, func(t *testing.T) {
			db, err := postgres.Connect(cachedStoreCtx, pgtest.GetConnectionString(t))
			require.NoError(t, err)
			t.Cleanup(db.Close)
			schema := pgx.Identifier{`cache "schema`}.Sanitize()
			table := pgx.Identifier{`cache "schema`, `table "name`}.Sanitize()
			column := pgx.Identifier{`key "name $rox_cache$`}.Sanitize()
			_, err = db.Exec(cachedStoreCtx, fmt.Sprintf("CREATE SCHEMA %s; CREATE TABLE %s (%s %s PRIMARY KEY, serialized bytea)", schema, table, column, keys.keyType))
			require.NoError(t, err)
			relation, err := installCacheNotifications(cachedStoreCtx, db, table)
			require.NoError(t, err)
			// Idempotent setup must not duplicate row or statement notifications.
			again, err := installCacheNotifications(cachedStoreCtx, db, table)
			require.NoError(t, err)
			require.Equal(t, relation, again)
			received := make(chan receivedCacheNotification, 20)
			ready, done := make(chan struct{}), make(chan struct{})
			ctx, cancel := context.WithCancel(cachedStoreCtx)
			t.Cleanup(func() { cancel(); awaitCacheTest(t, done) })
			listener := notify.NewListenerWithHooks(db, func(channel, payload string) { received <- receivedCacheNotification{channel, payload} },
				notify.LifecycleHooks{AfterListen: func(context.Context) error { close(ready); return nil }}, cacheNotificationChannel, "cache_test_barrier")
			go func() { listener.Listen(ctx); close(done) }()
			awaitCacheTest(t, ready)
			collect := func() []cacheNotification {
				t.Helper()
				require.NoError(t, notify.Notify(ctx, db, "cache_test_barrier", "committed"))
				var events []cacheNotification
				for {
					event := awaitCacheTest(t, received)
					if event.channel == "cache_test_barrier" {
						return events
					}
					require.Less(t, len(event.payload), 8000)
					var payload cacheNotification
					require.NoError(t, json.Unmarshal([]byte(event.payload), &payload))
					require.Equal(t, relation, payload.Relation)
					events = append(events, payload)
				}
			}
			_, err = db.Exec(ctx, fmt.Sprintf("INSERT INTO %s VALUES ($1, repeat('large protobuf', 10000)::bytea)", table), keys.first)
			require.NoError(t, err)
			events := collect()
			require.Len(t, events, 1)
			if name == "oversized key" {
				require.True(t, events[0].Full)
				require.Nil(t, events[0].Key)
			} else {
				require.Equal(t, keys.first, *events[0].Key)
			}
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			_, err = tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
			require.NoError(t, err)
			require.NoError(t, tx.Rollback(ctx))
			require.Empty(t, collect(), "rolled-back writes must emit no wakeups")
			_, err = db.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s = $1", table, column), keys.second)
			require.NoError(t, err)
			events = collect()
			require.Len(t, events, 2, "primary-key change must wake both the old and new key")
			require.Equal(t, keys.second, *events[1].Key)
			_, err = db.Exec(ctx, "TRUNCATE "+table)
			require.NoError(t, err)
			events = collect()
			require.Len(t, events, 1)
			require.True(t, events[0].Full)
		})
	}
}

func TestCacheNotificationSetupFailureRetries(t *testing.T) {
	db := pgtest.ForT(t)
	_, err := db.Exec(cachedStoreCtx, "DROP TABLE test_single_key_structs CASCADE")
	require.NoError(t, err)
	db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
	started := time.Now()
	store := newCachedStore(db)
	require.Less(t, time.Since(started), cacheSetupTimeout)
	retained, ok := store.(*retainedTestStore)
	require.True(t, ok)
	require.False(t, retained.CacheEnabled())
	_, _, err = store.Get(cachedStoreCtx, "missing")
	require.Error(t, err, "uncached fallback must propagate database failures")
	_, err = store.GetAllForSAC(cachedStoreCtx)
	require.Error(t, err, "SAC fallback must also propagate database failures")
}
