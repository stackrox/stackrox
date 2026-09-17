//go:build sql_integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cacheTestStore = Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]
type retainedTestStore = cachedStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct]

// This wrapper controls the boundary after a real snapshot has been read. It
// never replaces the PostgreSQL read or the production cache implementation.
type snapshotBarrierStore struct {
	cacheTestStore
	loaded  chan struct{}
	release chan struct{}
	failure atomic.Bool
}

func (s *snapshotBarrierStore) Walk(ctx context.Context, fn func(*storage.TestSingleKeyStruct) error) error {
	if err := s.cacheTestStore.Walk(ctx, fn); err != nil {
		return err
	}
	if s.loaded != nil {
		close(s.loaded)
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if s.failure.Load() {
		return errors.New("snapshot query failed after rows")
	}
	return nil
}

func retainedStore(t testing.TB, store cacheTestStore) *retainedTestStore {
	t.Helper()
	retained, ok := store.(*retainedTestStore)
	require.True(t, ok, "expected retained store, got %T", store)
	return retained
}

func awaitCacheTest[T any](t testing.TB, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(15 * time.Second):
		t.Fatal("timed out at readiness barrier")
		var zero T
		return zero
	}
}

func TestCacheSnapshotFailureAndGeneration(t *testing.T) {
	for name, invalidate := range map[string]func(*retainedTestStore, *snapshotBarrierStore){
		"query failure":    func(_ *retainedTestStore, barrier *snapshotBarrierStore) { barrier.failure.Store(true) },
		"new invalidation": func(store *retainedTestStore, _ *snapshotBarrierStore) { store.invalidateCache() },
	} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.ForT(t)
			store := retainedStore(t, newCachedStore(db))
			require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "old", 0)))
			barrier := &snapshotBarrierStore{cacheTestStore: store.underlyingStore, loaded: make(chan struct{}), release: make(chan struct{})}
			store.underlyingStore = barrier
			done := make(chan error, 1)
			go func() { done <- store.refreshCache(cachedStoreCtx, nil, true) }()
			awaitCacheTest(t, barrier.loaded)
			invalidate(store, barrier)
			require.NoError(t, barrier.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "latest", 1)))
			close(barrier.release)
			err := awaitCacheTest(t, done)
			if name == "query failure" {
				require.ErrorContains(t, err, "snapshot query failed")
			}
			require.False(t, store.CacheEnabled(), "an older scan cannot restore trust")
			got, found, err := store.Get(cachedStoreCtx, "row")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, "latest", got.GetName())
			require.NoError(t, store.refreshCache(cachedStoreCtx, []string{"row"}, false))
			require.False(t, store.CacheEnabled(), "key reload cannot repair an unknown gap")
			store.underlyingStore = barrier.cacheTestStore
			require.NoError(t, store.refreshCache(cachedStoreCtx, nil, true))
			require.True(t, store.CacheEnabled())
		})
	}
}

func TestCacheLocalWriteOverlappingSnapshot(t *testing.T) {
	db := pgtest.ForT(t)
	store := retainedStore(t, newCachedStore(db))
	require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "old", 0)))
	barrier := &snapshotBarrierStore{cacheTestStore: store.underlyingStore, loaded: make(chan struct{}), release: make(chan struct{})}
	store.underlyingStore = barrier
	scanned := make(chan error, 1)
	go func() { scanned <- store.refreshCache(cachedStoreCtx, nil, true) }()
	awaitCacheTest(t, barrier.loaded)
	written := make(chan error, 1)
	go func() { written <- store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "latest", 1)) }()
	close(barrier.release)
	require.NoError(t, awaitCacheTest(t, scanned))
	require.NoError(t, awaitCacheTest(t, written))
	got, found, err := store.Get(cachedStoreCtx, "row")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "latest", got.GetName())
}

func TestCacheCallerTransactionDoesNotInvertGateAndRowLock(t *testing.T) {
	db := pgtest.ForT(t)
	store := retainedStore(t, newCachedStore(db))
	require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "old", 0)))
	other := pgtest.ForTCustomPool(t, db.Config().ConnConfig.Database)
	t.Cleanup(other.Close)
	control := pgtest.ForTCustomPool(t, db.Config().ConnConfig.Database)
	t.Cleanup(control.Close)
	ctx, cancel := context.WithTimeout(cachedStoreCtx, 15*time.Second)
	defer cancel()
	tx, err := other.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	txCtx := postgres.ContextWithTx(ctx, tx)
	require.NoError(t, store.Upsert(txCtx, newCachedTestSingleKeyStruct("row", "tentative", 1)))
	written := make(chan error, 1)
	go func() { written <- store.Upsert(ctx, newCachedTestSingleKeyStruct("row", "committed", 2)) }()
	// Observe the ordinary writer waiting on the caller transaction's row
	// lock. It owns the mutation gate while PostgreSQL is blocked.
	require.Eventually(t, func() bool {
		var waiting bool
		err := control.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity
WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE 'INSERT INTO test_single_key_structs%')`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	scanned := make(chan error, 1)
	go func() { scanned <- store.refreshCache(ctx, nil, true) }()
	// This must bypass the gate even with an exclusive refresh queued.
	require.NoError(t, store.Upsert(txCtx, newCachedTestSingleKeyStruct("row", "tentative again", 3)))
	require.NoError(t, tx.Commit(ctx))
	require.NoError(t, awaitCacheTest(t, written))
	require.NoError(t, awaitCacheTest(t, scanned))
	got, found, err := store.Get(ctx, "row")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "committed", got.GetName())
}

func TestCacheUntrustedAndTransactionReadShortcuts(t *testing.T) {
	for mode, prepare := range map[string]func(*testing.T, *pgtest.TestPostgres, *retainedTestStore) context.Context{
		"untrusted": func(t *testing.T, db *pgtest.TestPostgres, store *retainedTestStore) context.Context {
			require.NoError(t, store.underlyingStore.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("new", "visible", 1)))
			store.invalidateCache()
			return cachedStoreCtx
		},
		"caller transaction": func(t *testing.T, db *pgtest.TestPostgres, store *retainedTestStore) context.Context {
			tx, err := db.Begin(cachedStoreCtx)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx.Rollback(cachedStoreCtx)) })
			ctx := postgres.ContextWithTx(cachedStoreCtx, tx)
			require.NoError(t, store.underlyingStore.Upsert(ctx, newCachedTestSingleKeyStruct("new", "visible", 1)))
			return ctx
		},
	} {
		t.Run(mode, func(t *testing.T) {
			db := pgtest.ForT(t)
			store := retainedStore(t, newCachedStore(db))
			ctx := prepare(t, db, store)
			for name, check := range map[string]func(*testing.T){
				"get": func(t *testing.T) {
					got, found, err := store.Get(ctx, "new")
					require.NoError(t, err)
					require.True(t, found)
					require.Equal(t, "visible", got.GetName())
				},
				"exists": func(t *testing.T) {
					found, err := store.Exists(ctx, "new")
					require.NoError(t, err)
					require.True(t, found)
				},
				"count": func(t *testing.T) {
					count, err := store.Count(ctx, nil)
					require.NoError(t, err)
					require.Equal(t, 1, count)
				},
				"many": func(t *testing.T) {
					got, missing, err := store.GetMany(ctx, []string{"new", "absent"})
					require.NoError(t, err)
					require.Len(t, got, 1)
					require.Equal(t, []int{1}, missing)
				},
				"ids": func(t *testing.T) {
					ids, err := store.GetIDs(ctx)
					require.NoError(t, err)
					require.Equal(t, []string{"new"}, ids)
				},
				"ids by query": func(t *testing.T) {
					ids, err := store.GetIDsByQuery(ctx, nil)
					require.NoError(t, err)
					require.Equal(t, []string{"new"}, ids)
				},
				"get by query": func(t *testing.T) {
					got, err := store.GetByQuery(ctx, nil)
					require.NoError(t, err)
					require.Len(t, got, 1)
				},
				"SAC": func(t *testing.T) {
					got, err := store.GetAllForSAC(sac.WithNoAccess(ctx))
					require.NoError(t, err)
					require.Len(t, got, 1)
				},
			} {
				t.Run(name, check)
			}
			for name, walk := range map[string]func(func(*storage.TestSingleKeyStruct) error) error{
				"walk":       func(fn func(*storage.TestSingleKeyStruct) error) error { return store.Walk(ctx, fn) },
				"walk query": func(fn func(*storage.TestSingleKeyStruct) error) error { return store.WalkByQuery(ctx, nil, fn) },
				"query fn":   func(fn func(*storage.TestSingleKeyStruct) error) error { return store.GetByQueryFn(ctx, nil, fn) },
			} {
				t.Run(name, func(t *testing.T) {
					var names []string
					require.NoError(t, walk(func(obj *storage.TestSingleKeyStruct) error { names = append(names, obj.GetName()); return nil }))
					require.Equal(t, []string{"visible"}, names)
				})
			}
			require.NoError(t, store.Delete(ctx, "new"))
			found, err := store.underlyingStore.Exists(ctx, "new")
			require.NoError(t, err)
			require.False(t, found, "delete must not short circuit on stale cache absence")
		})
	}
}

func TestCacheWalkCallbackCanMutateStore(t *testing.T) {
	db := pgtest.ForT(t)
	store := newCachedStore(db)
	require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "value", 1)))
	done := make(chan error, 1)
	go func() {
		done <- store.Walk(cachedStoreCtx, func(obj *storage.TestSingleKeyStruct) error { return store.Delete(cachedStoreCtx, obj.GetKey()) })
	}()
	require.NoError(t, awaitCacheTest(t, done))
}

func TestCacheObserverRetainsDeletionBeforeFirstUpsertDelivery(t *testing.T) {
	db := pgtest.ForT(t)
	store := retainedStore(t, newCachedStore(db))
	// Drive the real typed refresh/delivery boundaries explicitly to place a
	// completed local write after delivery but before the next DB snapshot.
	coordinator := &cacheCoordinator{ctx: cachedStoreCtx, connected: true}
	store.registration = &cacheRegistration{coordinator: coordinator, keys: make(map[string]struct{}), wake: make(chan struct{}, 1)}
	var deletions []*storage.TestSingleKeyStruct
	stop, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(_ context.Context, changes CacheChanges[*storage.TestSingleKeyStruct]) error {
		deletions = append(deletions, changes.Deleted...)
		return nil
	})
	require.NoError(t, err)
	defer stop()
	require.NoError(t, store.deliverCacheChanges(cachedStoreCtx))
	object := newCachedTestSingleKeyStruct("between", "previous object", 1)
	require.NoError(t, store.Upsert(cachedStoreCtx, object))
	require.NoError(t, store.underlyingStore.Delete(cachedStoreCtx, object.GetKey()))
	require.NoError(t, store.refreshCache(cachedStoreCtx, []string{object.GetKey()}, false))
	require.NoError(t, store.deliverCacheChanges(cachedStoreCtx))
	protoassert.ElementsMatch(t, []*storage.TestSingleKeyStruct{object}, deletions)
}

func TestCacheObserverDeletionBacklogIsBounded(t *testing.T) {
	db := pgtest.ForT(t)
	store := retainedStore(t, newCachedStore(db))
	store.registration = &cacheRegistration{
		coordinator: &cacheCoordinator{ctx: cachedStoreCtx, connected: true},
		keys:        make(map[string]struct{}), wake: make(chan struct{}, 1),
	}
	var deletions []*storage.TestSingleKeyStruct
	stop, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(_ context.Context, changes CacheChanges[*storage.TestSingleKeyStruct]) error {
		deletions = append(deletions, changes.Deleted...)
		return nil
	})
	require.NoError(t, err)
	defer stop()
	objects := make([]*storage.TestSingleKeyStruct, 0, cachePendingKeys+1)
	ids := make([]string, 0, cachePendingKeys+1)
	for i := range cachePendingKeys + 1 {
		id := fmt.Sprint(i)
		ids = append(ids, id)
		objects = append(objects, newCachedTestSingleKeyStruct(id, id, 1))
	}
	require.NoError(t, store.UpsertMany(cachedStoreCtx, objects))
	require.NoError(t, store.deliverCacheChanges(cachedStoreCtx))
	// Hold deliveries while a large local delete commits. Overflow must
	// preserve the remaining old cache row for full-reconcile cleanup.
	require.NoError(t, store.DeleteMany(cachedStoreCtx, ids))
	require.False(t, store.CacheEnabled())
	require.Len(t, store.removed, cachePendingKeys)
	require.Len(t, store.cache, 1)
	require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("while paused", "db only", 2)))
	require.Len(t, store.cache, 1, "untrusted local writes must not grow the observer backlog")
	count, err := store.Count(cachedStoreCtx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, count, "reads must reflect PostgreSQL while local cache application is paused")
	require.NoError(t, store.refreshCache(cachedStoreCtx, nil, true))
	require.NoError(t, store.deliverCacheChanges(cachedStoreCtx))
	protoassert.ElementsMatch(t, objects, deletions)
	require.True(t, store.CacheEnabled())
	require.Empty(t, store.removed)
}

func TestCoordinatedDecodeFailureRetriesFullSnapshot(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(cachedStoreCtx, db.DB)
	store := retainedStore(t, newCachedStore(db))
	object := newCachedTestSingleKeyStruct("row", "valid", 1)
	require.NoError(t, store.Upsert(cachedStoreCtx, object))
	_, err := db.Exec(cachedStoreCtx, "UPDATE test_single_key_structs SET serialized = $1 WHERE key = 'row'", []byte{0x0f})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return !store.CacheEnabled() }, 5*time.Second, 10*time.Millisecond)
	_, _, err = store.Get(cachedStoreCtx, "row")
	require.Error(t, err, "decode errors must not appear as cache misses")
	_, err = store.GetAllForSAC(cachedStoreCtx)
	require.Error(t, err)
	// Failure preserves the prior complete map, but no shortcut may use it.
	concurrency.WithRLock(&store.cacheLock, func() { protoassert.Equal(t, object, store.cache["row"]) })
	require.NoError(t, store.underlyingStore.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "repaired", 2)))
	require.Eventually(t, func() bool {
		if !store.CacheEnabled() {
			return false
		}
		got, _, err := store.Get(cachedStoreCtx, "row")
		return err == nil && got.GetName() == "repaired"
	}, 5*time.Second, 10*time.Millisecond)
}

func TestCacheFailedTransactionPropagatesReadErrors(t *testing.T) {
	db := pgtest.ForT(t)
	store := newCachedStore(db)
	require.NoError(t, store.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("row", "value", 1)))
	tx, err := db.Begin(cachedStoreCtx)
	require.NoError(t, err)
	defer func() { require.NoError(t, tx.Rollback(cachedStoreCtx)) }()
	ctx := postgres.ContextWithTx(cachedStoreCtx, tx)
	_, err = tx.Exec(ctx, "SELECT 1/0")
	require.Error(t, err)
	_, _, err = store.Get(ctx, "row")
	require.Error(t, err)
	_, err = store.GetAllForSAC(ctx)
	require.Error(t, err)
	_, err = store.Count(ctx, nil)
	require.Error(t, err)
}

type batchCountingStore struct {
	cacheTestStore
	reloads int
}

func (s *batchCountingStore) GetMany(ctx context.Context, ids []string) ([]*storage.TestSingleKeyStruct, []int, error) {
	s.reloads++
	return s.cacheTestStore.GetMany(ctx, ids)
}

func BenchmarkCacheBatchedRefresh(b *testing.B) {
	db := pgtest.ForT(b)
	store := retainedStore(b, newCachedStore(db))
	objects := make([]*storage.TestSingleKeyStruct, 0, 200)
	keys := make([]string, 0, 200)
	for i := range 200 {
		key := fmt.Sprintf("key-%d", i)
		keys = append(keys, key)
		objects = append(objects, newCachedTestSingleKeyStruct(key, key, int64(i)))
	}
	require.NoError(b, store.UpsertMany(cachedStoreCtx, objects))
	counter := &batchCountingStore{cacheTestStore: store.underlyingStore}
	store.underlyingStore = counter
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := store.refreshCache(cachedStoreCtx, keys, false); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(counter.reloads)/float64(b.N), "reloads/batch")
	b.ReportMetric(float64(len(keys)), "keys/batch")
}

func TestCacheObserverInitialDeliveryAndDeletionRetry(t *testing.T) {
	db := pgtest.ForT(t)
	db.DB = WithCacheCoordination(context.Background(), db.DB)
	store := retainedStore(t, newCachedStore(db))
	other := retainedStore(t, newCachedStore(db))
	object := newCachedTestSingleKeyStruct("row", "previous", 1)
	require.NoError(t, store.Upsert(cachedStoreCtx, object))
	initial := make(chan CacheChanges[*storage.TestSingleKeyStruct], 8)
	failed := make(chan struct{}, 1)
	retried := make(chan CacheChanges[*storage.TestSingleKeyStruct], 1)
	var attempts atomic.Int32
	var reentered atomic.Bool
	unsubscribe, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(ctx context.Context, changes CacheChanges[*storage.TestSingleKeyStruct]) error {
		// Exercise read, mutation and subscription reentry without any of the
		// cache, mutation or observer registry locks on the callback stack.
		if reentered.CompareAndSwap(false, true) {
			if err := store.Upsert(ctx, object); err != nil {
				return err
			}
			stop, err := ObserveCache[*storage.TestSingleKeyStruct](store, func(context.Context, CacheChanges[*storage.TestSingleKeyStruct]) error { return nil })
			if err != nil {
				return err
			}
			stop()
		}
		_, err := store.GetIDs(ctx)
		if err != nil {
			return err
		}
		if len(changes.Deleted) == 0 {
			select {
			case initial <- changes:
			default:
			}
			return nil
		}
		if attempts.Add(1) == 1 {
			changes.Deleted[0].Name = "callback owns its clone"
			failed <- struct{}{}
			return errors.New("derived cleanup unavailable")
		}
		retried <- changes
		return nil
	})
	require.NoError(t, err)
	defer unsubscribe()
	first := awaitCacheTest(t, initial)
	require.True(t, first.Full)
	protoassert.ElementsMatch(t, []*storage.TestSingleKeyStruct{object}, first.Upserts)
	require.NoError(t, store.Delete(cachedStoreCtx, "row"))
	awaitCacheTest(t, failed)
	require.Eventually(t, func() bool { return !store.CacheEnabled() }, time.Second, time.Millisecond)
	// Other registrations keep progressing during the failed delivery.
	require.NoError(t, other.Upsert(cachedStoreCtx, newCachedTestSingleKeyStruct("other", "independent", 2)))
	require.Eventually(t, func() bool {
		return other.CacheEnabled() && concurrency.WithRLock1(&other.cacheLock, func() bool { return other.cache["row"] == nil })
	}, time.Second, 10*time.Millisecond)
	changes := awaitCacheTest(t, retried)
	protoassert.ElementsMatch(t, []*storage.TestSingleKeyStruct{object}, changes.Deleted)
	require.Eventually(t, store.CacheEnabled, 5*time.Second, 10*time.Millisecond)
	unsubscribe()
	assert.True(t, reentered.Load())
}
