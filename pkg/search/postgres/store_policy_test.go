//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPolicyTestStore(db postgres.DB, global bool) Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct] {
	if global {
		return NewGloballyScopedGenericStoreWithCache[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
			db, schema.TestSingleKeyStructsSchema, pkGetter, insertIntoTestSingleKeyStructs, copyFromTestSingleKeyStructs,
			doNothingDurationTimeSetter, doNothingDurationTimeSetter, doNothingDurationTimeSetter, resources.Administration, nil, nil)
	}
	return NewGenericStoreWithCache[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		db, schema.TestSingleKeyStructsSchema, pkGetter, insertIntoTestSingleKeyStructs, copyFromTestSingleKeyStructs,
		doNothingDurationTimeSetter, doNothingDurationTimeSetter, doNothingDurationTimeSetter,
		globallyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace), resources.Namespace, nil, nil)
}

func TestWorkerStoreSeesIndependentWrites(t *testing.T) {
	for name, global := range map[string]bool{"scoped": false, "global": true} {
		t.Run(name, func(t *testing.T) {
			testDB := pgtest.ForT(t)
			worker := newPolicyTestStore(WithoutStoreCache(testDB.DB), global)
			central := newPolicyTestStore(testDB.DB, global)
			ctx := sac.WithAllAccess(context.Background())
			for stage, obj := range []*storage.TestSingleKeyStruct{
				{Key: "later", Name: "inserted", Int64: 1},
				{Key: "later", Name: "updated", Int64: 2},
				nil,
			} {
				if obj == nil {
					require.NoError(t, central.Delete(ctx, "later"))
				} else {
					require.NoError(t, central.Upsert(ctx, obj))
				}
				var expected []*storage.TestSingleKeyStruct
				var expectedIDs []string
				var missing []int
				if obj != nil {
					expected = []*storage.TestSingleKeyStruct{obj}
					expectedIDs = []string{"later"}
				} else {
					missing = []int{0}
				}
				got, found, err := worker.Get(ctx, "later")
				require.NoError(t, err)
				assert.Equal(t, obj != nil, found, "stage %d", stage)
				protoassert.Equal(t, obj, got)
				exists, err := worker.Exists(ctx, "later")
				require.NoError(t, err)
				assert.Equal(t, obj != nil, exists)
				many, absent, err := worker.GetMany(ctx, []string{"later"})
				require.NoError(t, err)
				protoassert.ElementsMatch(t, expected, many)
				assert.ElementsMatch(t, missing, absent)
				count, err := worker.Count(ctx, search.EmptyQuery())
				require.NoError(t, err)
				assert.Equal(t, len(expected), count)
				ids, err := worker.GetIDs(ctx)
				require.NoError(t, err)
				assert.ElementsMatch(t, expectedIDs, ids)
				ids, err = worker.GetIDsByQuery(ctx, search.EmptyQuery())
				require.NoError(t, err)
				assert.ElementsMatch(t, expectedIDs, ids)
				results, err := worker.Search(ctx, search.EmptyQuery())
				require.NoError(t, err)
				assert.ElementsMatch(t, expectedIDs, search.ResultsToIDs(results))
				byQuery, err := worker.GetByQuery(ctx, search.EmptyQuery())
				require.NoError(t, err)
				protoassert.ElementsMatch(t, expected, byQuery)
				for walkName, walk := range map[string]func(func(*storage.TestSingleKeyStruct) error) error{
					"Walk": func(fn func(*storage.TestSingleKeyStruct) error) error { return worker.Walk(ctx, fn) },
					"WalkByQuery": func(fn func(*storage.TestSingleKeyStruct) error) error {
						return worker.WalkByQuery(ctx, search.EmptyQuery(), fn)
					},
					"GetByQueryFn": func(fn func(*storage.TestSingleKeyStruct) error) error {
						return worker.GetByQueryFn(ctx, search.EmptyQuery(), fn)
					},
				} {
					var walked []*storage.TestSingleKeyStruct
					require.NoError(t, walk(func(obj *storage.TestSingleKeyStruct) error { walked = append(walked, obj); return nil }), walkName)
					protoassert.ElementsMatch(t, expected, walked)
				}
				sacObjects, err := worker.GetAllForSAC(sac.WithNoAccess(ctx))
				require.NoError(t, err)
				protoassert.ElementsMatch(t, expected, sacObjects)
			}

			for operation, deleteRow := range map[string]func(context.Context, string) error{
				"Delete":     worker.Delete,
				"DeleteMany": func(ctx context.Context, id string) error { return worker.DeleteMany(ctx, []string{id}) },
				"PruneMany":  func(ctx context.Context, id string) error { return worker.PruneMany(ctx, []string{id}) },
			} {
				require.NoError(t, central.Upsert(ctx, &storage.TestSingleKeyStruct{Key: operation, Name: operation}))
				require.NoError(t, deleteRow(ctx, operation))
				_, found, err := newStore(testDB).Get(ctx, operation)
				require.NoError(t, err)
				assert.False(t, found, "%s must delete a row absent at startup", operation)
			}
		})
	}
}

func TestUncachedSACReadDoesNotPanic(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	ctx := sac.WithAllAccess(context.Background())
	require.NoError(t, store.Upsert(ctx, &storage.TestSingleKeyStruct{Key: "sac", Name: "visible to SAC"}))
	objects, err := store.GetAllForSAC(sac.WithNoAccess(ctx))
	require.NoError(t, err)
	protoassert.ElementsMatch(t, []*storage.TestSingleKeyStruct{{Key: "sac", Name: "visible to SAC"}}, objects)
}

func TestWorkerStoreReadFailures(t *testing.T) {
	for name, global := range map[string]bool{"scoped": false, "global": true} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.ForT(t).DB
			worker := newPolicyTestStore(WithoutStoreCache(db), global)
			db.Close()
			ctx := sac.WithAllAccess(context.Background())
			for operation, read := range map[string]func() error{
				"Get":     func() error { _, _, err := worker.Get(ctx, "missing"); return err },
				"Exists":  func() error { _, err := worker.Exists(ctx, "missing"); return err },
				"GetMany": func() error { _, _, err := worker.GetMany(ctx, []string{"missing"}); return err },
				"Count":   func() error { _, err := worker.Count(ctx, search.EmptyQuery()); return err },
				"GetIDs":  func() error { _, err := worker.GetIDs(ctx); return err },
				"Walk":    func() error { return worker.Walk(ctx, func(*storage.TestSingleKeyStruct) error { return nil }) },
				"SAC":     func() error { objects, err := worker.GetAllForSAC(ctx); assert.Nil(t, objects); return err },
			} {
				t.Run(operation, func(t *testing.T) { assert.Error(t, read()) })
			}
		})
	}
}

// Other DB policies expose Unwrap so cache policy survives either wrapper order.
type wrappedPolicyTestDB struct {
	postgres.DB
}

func (db wrappedPolicyTestDB) Unwrap() postgres.DB {
	return db.DB
}

func TestStoreCachePolicyComposition(t *testing.T) {
	for name, wrap := range map[string]func(postgres.DB) postgres.DB{
		"outer opt-out":    func(db postgres.DB) postgres.DB { return WithoutStoreCache(wrappedPolicyTestDB{DB: db}) },
		"inner opt-out":    func(db postgres.DB) postgres.DB { return wrappedPolicyTestDB{DB: WithoutStoreCache(db)} },
		"repeated opt-out": func(db postgres.DB) postgres.DB { return WithoutStoreCache(WithoutStoreCache(db)) },
	} {
		t.Run(name, func(t *testing.T) {
			for constructor, global := range map[string]bool{"scoped": false, "global": true} {
				t.Run(constructor, func(t *testing.T) {
					db := pgtest.ForT(t).DB
					worker := newPolicyTestStore(wrap(db), global)
					central := newPolicyTestStore(db, global)
					writer := newPolicyTestStore(db, global)
					ctx := sac.WithAllAccess(context.Background())
					require.NoError(t, writer.Upsert(ctx, &storage.TestSingleKeyStruct{Key: "later", Name: "later"}))
					ids, err := worker.GetIDs(ctx)
					require.NoError(t, err)
					assert.Equal(t, []string{"later"}, ids)
					ids, err = central.GetIDs(ctx)
					require.NoError(t, err)
					assert.Empty(t, ids, "ordinary Central keeps its cache on the same DB")
				})
			}
		})
	}
}

func TestStoreSACReadFailures(t *testing.T) {
	for constructor, global := range map[string]bool{"scoped": false, "global": true} {
		t.Run(constructor, func(t *testing.T) {
			db := pgtest.ForT(t).DB
			worker := newPolicyTestStore(WithoutStoreCache(db), global)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			objects, err := worker.GetAllForSAC(ctx)
			require.Error(t, err)
			assert.Nil(t, objects)

			conn, err := db.Acquire(context.Background())
			require.NoError(t, err)
			tx, txCtx, err := conn.Begin(context.Background())
			require.NoError(t, err)
			_, err = tx.Exec(txCtx, "SELECT 1 / 0")
			require.Error(t, err)
			objects, err = worker.GetAllForSAC(txCtx)
			require.Error(t, err, "a failed PostgreSQL transaction must not look like an empty SAC scope")
			assert.Nil(t, objects)
			require.NoError(t, tx.Rollback(context.Background()))
			conn.Release()

			db.Close()
			// Failed initial population must also support the error-returning SAC API.
			fallback := newPolicyTestStore(db, global)
			objects, err = fallback.GetAllForSAC(context.Background())
			require.Error(t, err)
			assert.Nil(t, objects)
		})
	}
}
