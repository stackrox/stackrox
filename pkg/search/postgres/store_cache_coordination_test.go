//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/require"
)

type initialSnapshotDB struct {
	postgres.DB
	begins  atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (db *initialSnapshotDB) Begin(ctx context.Context) (*postgres.Tx, error) {
	// Trigger installation is the first transaction; initial Walk is second.
	if db.begins.Add(1) == 2 {
		close(db.started)
		select {
		case <-db.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return db.DB.Begin(ctx)
}

func TestCoordinatedInitializationOrdering(t *testing.T) {
	db := pgtest.ForT(t)
	writer := newCachedStore(&pgtest.TestPostgres{DB: WithoutStoreCache(db.DB)})
	barrier := &initialSnapshotDB{DB: db.DB, started: make(chan struct{}), release: make(chan struct{})}
	db.DB = WithCacheCoordination(cachedStoreCtx, barrier)
	constructed := make(chan cacheTestStore, 1)
	go func() { constructed <- newCachedStore(db) }()
	awaitCacheTest(t, barrier.started)
	require.True(t, cacheCoordinatorFor(db.DB).isConnected(), "LISTEN must commit before snapshot begins")
	require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("during setup", "initialized", 1)))
	close(barrier.release)
	store := retainedStore(t, awaitCacheTest(t, constructed))
	require.Eventually(t, store.CacheEnabled, 5*time.Second, 10*time.Millisecond)
	got, found, err := store.Get(cachedStoreCtx, "during setup")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "initialized", got.GetName())
}

func TestCoordinatedPeriodicReconciliation(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
	store := retainedStore(t, newCachedStore(db))
	initialized := make(chan struct{}, 1)
	unsubscribe, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(context.Context, CacheChanges[*storage.TestSingleKeyStruct]) error {
		select {
		case initialized <- struct{}{}:
		default:
		}
		return nil
	})
	require.NoError(t, err)
	defer unsubscribe()
	awaitCacheTest(t, initialized)
	_, err = db.Exec(cachedStoreCtx, "ALTER TABLE test_single_key_structs DISABLE TRIGGER rox_cache_row_change")
	require.NoError(t, err)
	require.NoError(t, store.underlyingStore.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("silent", "periodic", 1)))
	// No wakeup is emitted. The production periodic ticker must recover it.
	require.Eventually(t, func() bool {
		return store.CacheEnabled() && concurrency.WithRLock1(&store.cacheLock, func() bool {
			return store.cache["silent"] != nil
		})
	}, cacheReconcileInterval+15*time.Second, 100*time.Millisecond)
}

func TestCoordinatedReconnectAndClose(t *testing.T) {
	db := pgtest.ForT(t)
	coordinated := WithCacheCoordination(cachedStoreCtx, db.DB)
	db.DB = coordinated
	store := retainedStore(t, newCachedStore(db))
	second := retainedStore(t, newCachedStore(db))
	coordinator := cacheCoordinatorFor(coordinated)
	require.Same(t, coordinated, WithCacheCoordination(cachedStoreCtx, coordinated))
	var pids []int
	rows, err := db.Query(cachedStoreCtx, `SELECT pid FROM pg_stat_activity
WHERE datname = current_database() AND query = 'LISTEN "rox_retained_cache"'`)
	require.NoError(t, err)
	for rows.Next() {
		var pid int
		require.NoError(t, rows.Scan(&pid))
		pids = append(pids, pid)
	}
	require.NoError(t, rows.Err())
	rows.Close()
	require.Len(t, pids, 1, "all registrations share one dedicated listener")
	var terminated bool
	require.NoError(t, db.QueryRow(cachedStoreCtx, "SELECT pg_terminate_backend($1)", pids[0]).Scan(&terminated))
	require.True(t, terminated)
	require.Eventually(t, func() bool { return !coordinator.isConnected() && !store.CacheEnabled() && !second.CacheEnabled() },
		3*time.Second, 10*time.Millisecond)
	require.NoError(t, store.underlyingStore.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("downtime", "recovered", 1)))
	got, found, err := second.Get(cachedStoreCtx, "downtime")
	require.NoError(t, err)
	require.True(t, found, "untrusted cache must fall back during downtime")
	require.Equal(t, "recovered", got.GetName())
	require.Eventually(t, func() bool {
		return store.CacheEnabled() && second.CacheEnabled() && concurrency.WithRLock1(&second.cacheLock, func() bool {
			return second.cache["downtime"] != nil
		})
	}, 12*time.Second, 10*time.Millisecond, "reconnect must reconcile the notification gap")
	closed := make(chan struct{})
	go func() { coordinated.Close(); close(closed) }()
	awaitCacheTest(t, closed)
	require.False(t, store.CacheEnabled())
}

func TestCoordinatedInitialRefreshFailureRecovers(t *testing.T) {
	db := pgtest.ForT(t)
	writer := newCachedStore(&pgtest.TestPostgres{DB: WithoutStoreCache(db.DB)})
	require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("initial", "valid", 1)))
	_, err := db.Exec(cachedStoreCtx, "UPDATE test_single_key_structs SET serialized = $1", []byte{0x0f})
	require.NoError(t, err)
	db.DB = WithCacheCoordination(context.Background(), db.DB)
	store := retainedStore(t, newCachedStore(db))
	require.False(t, store.CacheEnabled())
	_, err = store.GetAllForSAC(cachedStoreCtx)
	require.Error(t, err, "failed initialization must not appear as an empty SAC scope")
	ready := make(chan CacheChanges[*storage.TestSingleKeyStruct], 1)
	stop, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(_ context.Context, changes CacheChanges[*storage.TestSingleKeyStruct]) error {
		select {
		case ready <- changes:
		default:
		}
		return nil
	})
	require.NoError(t, err, "subscription must work even while initial cache trust is false")
	defer stop()
	require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("initial", "repaired", 2)))
	changes := awaitCacheTest(t, ready)
	require.True(t, changes.Full)
	require.Len(t, changes.Upserts, 1)
	require.Equal(t, "repaired", changes.Upserts[0].GetName())
	require.True(t, store.CacheEnabled())
}

func TestCoordinatedCloseDuringInitialization(t *testing.T) {
	db := pgtest.ForT(t)
	barrier := &initialSnapshotDB{DB: db.DB, started: make(chan struct{}), release: make(chan struct{})}
	db.DB = WithCacheCoordination(context.Background(), barrier)
	constructed := make(chan cacheTestStore, 1)
	go func() { constructed <- newCachedStore(db) }()
	awaitCacheTest(t, barrier.started)
	closed := make(chan struct{})
	go func() { db.Close(); close(closed) }()
	awaitCacheTest(t, closed)
	store := retainedStore(t, awaitCacheTest(t, constructed))
	require.False(t, store.CacheEnabled())
	_, err := store.GetAllForSAC(cachedStoreCtx)
	require.Error(t, err)
}

func TestCoordinatedCloseCancelsAndJoinsObserver(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(context.Background(), db.DB)
	store := newCachedStore(db)
	entered, exited := make(chan struct{}), make(chan struct{})
	stop, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(ctx context.Context, _ CacheChanges[*storage.TestSingleKeyStruct]) error {
		close(entered)
		<-ctx.Done()
		close(exited)
		return ctx.Err()
	})
	require.NoError(t, err)
	defer stop()
	awaitCacheTest(t, entered)
	closed := make(chan struct{})
	go func() { db.Close(); close(closed) }()
	awaitCacheTest(t, closed)
	select {
	case <-exited:
	default:
		t.Fatal("Close returned before observer exited")
	}
	_, err = ObserveCache[*storage.TestSingleKeyStruct](store, func(context.Context, CacheChanges[*storage.TestSingleKeyStruct]) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
}

func TestCoordinatedPolicyAndSetupFailure(t *testing.T) {
	for name, wrap := range map[string]func(postgres.DB) postgres.DB{
		"worker outside": func(db postgres.DB) postgres.DB { return WithoutStoreCache(WithCacheCoordination(cachedStoreCtx, db)) },
		"worker inside":  func(db postgres.DB) postgres.DB { return WithCacheCoordination(cachedStoreCtx, WithoutStoreCache(db)) },
		"cancelled setup": func(db postgres.DB) postgres.DB {
			ctx, cancel := context.WithCancel(cachedStoreCtx)
			cancel()
			return WithCacheCoordination(ctx, db)
		},
	} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.ForT(t)
			db.DB = wrap(db.DB)
			store := newCachedStore(db)
			cache, ok := store.(interface{ CacheEnabled() bool })
			require.True(t, ok)
			require.False(t, cache.CacheEnabled())
			_, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(context.Context, CacheChanges[*storage.TestSingleKeyStruct]) error { return nil })
			require.Error(t, err)
			require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "uncached", 1)))
		})
	}
}

func TestCoordinatedConcurrentRegistration(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
	stores := make(chan cacheTestStore, 6)
	for i := range 6 {
		go func() { stores <- newPolicyTestStore(db.DB, i%2 == 0) }()
	}
	retained := make([]*retainedTestStore, 0, 6)
	for range 6 {
		retained = append(retained, retainedStore(t, awaitCacheTest(t, stores)))
	}
	require.NoError(t, retained[0].Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("shared", "registered", 1)))
	for _, store := range retained {
		require.Eventually(t, func() bool {
			return store.CacheEnabled() && concurrency.WithRLock1(&store.cacheLock, func() bool { return store.cache["shared"] != nil })
		}, 5*time.Second, 10*time.Millisecond)
	}
}

func TestCoordinatedBatchedWritesAndTruncate(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
	reader := retainedStore(t, newCachedStore(db))
	objects := make([]*storage.TestSingleKeyStruct, 0, 200)
	for i := range 200 {
		objects = append(objects, newCachedTestSingleKeyStruct(fmt.Sprint(i), fmt.Sprintf("bulk %d", i), int64(i)))
	}
	require.NoError(t, reader.underlyingStore.UpsertMany(cachedStoreCtx, objects))
	require.Eventually(t, func() bool {
		return reader.CacheEnabled() && concurrency.WithRLock1(&reader.cacheLock, func() bool { return len(reader.cache) == 200 })
	}, 5*time.Second, 10*time.Millisecond)
	_, err := db.Exec(cachedStoreCtx, "TRUNCATE test_single_key_structs CASCADE")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return reader.CacheEnabled() && concurrency.WithRLock1(&reader.cacheLock, func() bool { return len(reader.cache) == 0 })
	}, 5*time.Second, 10*time.Millisecond)
}

// A second retained instance must follow committed writes made through another
// store; neither instance can rely on being the only writer of its table.
func TestCoordinatedIndependentCaches(t *testing.T) {
	for name, change := range map[string]func(*testing.T, Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]){
		"insert": func(t *testing.T, writer Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]) {
			require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("external", "new", 1)))
		},
		"update": func(t *testing.T, writer Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]) {
			require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("existing", "new", 1)))
		},
		"delete": func(t *testing.T, writer Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]) {
			require.NoError(t, writer.Delete(cachedStoreCtx, "existing"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.ForT(t)
			db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
			writer := newCachedStore(db)
			require.NoError(t, writer.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("existing", "old", 0)))
			reader := newCachedStore(db)
			change(t, writer)
			want, err := writer.GetAllForSAC(cachedStoreCtx)
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				got, err := reader.GetAllForSAC(cachedStoreCtx)
				if err != nil || len(got) != len(want) {
					return false
				}
				byKey := make(map[string]*storage.TestSingleKeyStruct, len(got))
				for _, obj := range got {
					byKey[obj.GetKey()] = obj
				}
				for _, obj := range want {
					if !obj.EqualVT(byKey[obj.GetKey()]) {
						return false
					}
				}
				return true
			}, 3*time.Second, 10*time.Millisecond, "independent cache missed committed "+name)
		})
	}
}

func TestCachedCallerTransactionRollback(t *testing.T) {
	db := pgtest.ForT(t)
	store := newCachedStore(db)
	other := pgtest.ForTCustomPool(t, db.Config().ConnConfig.Database)
	t.Cleanup(other.Close)
	tx, err := other.Begin(cachedStoreCtx)
	require.NoError(t, err)
	txCtx := postgres.ContextWithTx(cachedStoreCtx, tx)
	require.NoError(t, store.Upsert(txCtx, newCachedTestSingleKeyStruct("tentative", "uncommitted", 1)))
	_, found, err := store.Get(txCtx, "tentative")
	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, tx.Rollback(cachedStoreCtx))
	_, found, err = store.Get(cachedStoreCtx, "tentative")
	require.NoError(t, err)
	require.False(t, found, "rollback must not publish tentative cache objects")
}
