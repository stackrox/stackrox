//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cachedStoreCtx = sac.WithAllAccess(context.Background())
)

func TestNewCachedStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, newCachedStore(testDB))
}

func TestNewGenericCachedStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, NewGenericStoreWithCache[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetterForCache,
		insertIntoTestSingleKeyStructsWithCache,
		copyFromTestSingleKeyStructsWithCache,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		globallyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	))
}

func TestNewGloballyScopedGenericCachedStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, NewGloballyScopedGenericStoreWithCache[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetterForCache,
		insertIntoTestSingleKeyStructsWithCache,
		copyFromTestSingleKeyStructsWithCache,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		resources.Namespace,
	))
}

func TestCachedUpsert(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	key := "TestUpsert"
	name := "Test Upsert"
	testObject := newCachedTestSingleKeyStruct(key, name, int64(1))

	objBefore, foundBefore, errBefore := store.Get(cachedStoreCtx, key)
	assert.Nil(t, objBefore)
	assert.False(t, foundBefore)
	assert.NoError(t, errBefore)

	assert.NoError(t, store.Upsert(cachedStoreCtx, testObject))

	objAfter, foundAfter, errAfter := store.Get(cachedStoreCtx, key)
	protoassert.Equal(t, testObject, objAfter)
	assert.True(t, foundAfter)
	assert.NoError(t, errAfter)
}

func TestCachedUpsertMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)

	testObjects := sampleCachedTestSingleKeyStructArray("UpsertMany")

	for _, obj := range testObjects {
		objBefore, foundBefore, errBefore := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		assert.Nil(t, objBefore)
		assert.False(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	for _, obj := range testObjects {
		objAfter, foundAfter, errAfter := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		protoassert.Equal(t, obj, objAfter)
		assert.True(t, foundAfter)
		assert.NoError(t, errAfter)
	}
}

func TestCachedDelete(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	key := "TestDelete"
	name := "Test Delete"
	testObject := newCachedTestSingleKeyStruct(key, name, int64(1))
	require.NoError(t, store.Upsert(cachedStoreCtx, testObject))

	objBefore, foundBefore, errBefore := store.Get(cachedStoreCtx, key)
	protoassert.Equal(t, testObject, objBefore)
	require.True(t, foundBefore)
	require.NoError(t, errBefore)

	assert.NoError(t, store.Delete(cachedStoreCtx, key))

	objAfter, foundAfter, errAfter := store.Get(cachedStoreCtx, key)
	require.Nil(t, objAfter)
	require.False(t, foundAfter)
	require.NoError(t, errAfter)

	missingKey := "TestDeleteMissingKey"

	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(cachedStoreCtx, missingKey)
	require.Nil(t, missingObjBefore)
	require.False(t, missingFoundBefore)
	require.NoError(t, missingErrBefore)

	assert.NoError(t, store.Delete(cachedStoreCtx, missingKey))

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(cachedStoreCtx, missingKey)
	require.Nil(t, missingObjAfter)
	require.False(t, missingFoundAfter)
	require.NoError(t, missingErrAfter)
}

func TestCachedDeleteMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	objectBatch := sampleCachedTestSingleKeyStructArray("DeleteMany")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, objectBatch))

	identifiersToRemove := make([]string, 0, len(objectBatch)+1)
	for _, obj := range objectBatch {
		key := pkGetterForCache(obj)
		identifiersToRemove = append(identifiersToRemove, key)
		// ensure object is in DB before call to remove
		objBefore, foundBefore, errBefore := store.Get(cachedStoreCtx, key)
		protoassert.Equal(t, obj, objBefore)
		assert.True(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	missingKey := "TestDeleteManyMissingKey"
	identifiersToRemove = append(identifiersToRemove, missingKey)
	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(cachedStoreCtx, missingKey)
	assert.Nil(t, missingObjBefore)
	assert.False(t, missingFoundBefore)
	assert.NoError(t, missingErrBefore)

	assert.NoError(t, store.DeleteMany(cachedStoreCtx, identifiersToRemove))

	for _, obj := range objectBatch {
		key := pkGetterForCache(obj)
		// ensure object is NOT in DB after call to remove
		objAfter, foundAfter, errAfter := store.Get(cachedStoreCtx, key)
		assert.Nil(t, objAfter)
		assert.False(t, foundAfter)
		assert.NoError(t, errAfter)
	}

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(cachedStoreCtx, missingKey)
	assert.Nil(t, missingObjAfter)
	assert.False(t, missingFoundAfter)
	assert.NoError(t, missingErrAfter)
}

func TestCachedExists(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	key := "TestExists"
	name := "Test Exists"
	testObject := newCachedTestSingleKeyStruct(key, name, int64(9))

	require.NoError(t, store.Upsert(cachedStoreCtx, testObject))

	missingKey := "TestExistsMissingKey"

	foundExisting, errExisting := store.Exists(cachedStoreCtx, key)
	assert.True(t, foundExisting)
	assert.NoError(t, errExisting)

	foundMissing, errMissing := store.Exists(cachedStoreCtx, missingKey)
	assert.False(t, foundMissing)
	assert.NoError(t, errMissing)
}

func TestCachedCount(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	firstCount, err1 := store.Count(cachedStoreCtx, nil)
	assert.Equal(t, 0, firstCount)
	assert.NoError(t, err1)

	testObject1 := newCachedTestSingleKeyStruct("TestCount", "Test Count", int64(256))
	assert.NoError(t, store.Upsert(cachedStoreCtx, testObject1))

	secondCount, err2 := store.Count(cachedStoreCtx, nil)
	assert.Equal(t, 1, secondCount)
	assert.NoError(t, err2)

	secondCount, err2 = store.Count(cachedStoreCtx, search.EmptyQuery())
	assert.Equal(t, 1, secondCount)
	assert.NoError(t, err2)

	supplementaryObjects := sampleCachedTestSingleKeyStructArray("Count")
	assert.NoError(t, store.UpsertMany(cachedStoreCtx, supplementaryObjects))

	thirdCount, err3 := store.Count(cachedStoreCtx, nil)
	assert.Equal(t, 1+len(supplementaryObjects), thirdCount)
	assert.NoError(t, err3)
}

func TestCachedWalk(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("Walk")
	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	injectedNames := make([]string, 0, len(testObjects))
	for _, obj := range testObjects {
		injectedNames = append(injectedNames, obj.GetName())
	}

	walkedNames := make([]string, 0, len(testObjects))
	walkedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))

	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedNames = append(walkedNames, obj.Name)
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	assert.NoError(t, store.Walk(cachedStoreCtx, walkFn))

	assert.ElementsMatch(t, walkedNames, injectedNames)
	protoassert.ElementsMatch(t, testObjects, walkedObjects)
}

func TestCachedWalkContextCancelation(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("Walk")
	err := store.UpsertMany(cachedStoreCtx, testObjects)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(cachedStoreCtx)
	cancel()
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		return nil
	}
	err = store.Walk(ctx, walkFn)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestCachedGetByQueryDoesNotModifyTheObject(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("WalkByQuery")
	err := store.UpsertMany(cachedStoreCtx, testObjects)
	require.NoError(t, err)

	query2 := getCachedMatchFieldQuery("Test Name", "Test WalkByQuery 2")
	query4 := getCachedMatchFieldQuery("Test Key", "TestWalkByQuery4")
	query := getCachedDisjunctionQuery(query2, query4)

	walkedNames := make([]string, 0, len(testObjects))
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedNames = append(walkedNames, obj.Name)
		obj.Name = "changed"
		return nil
	}
	err = store.GetByQueryFn(cachedStoreCtx, query, walkFn)
	require.NoError(t, err)

	expectedNames := []string{
		"Test WalkByQuery 2",
		"Test WalkByQuery 4",
	}
	assert.ElementsMatch(t, expectedNames, walkedNames)

	walkedNames = make([]string, 0, len(testObjects))
	err = store.GetByQueryFn(cachedStoreCtx, nil, walkFn)
	assert.NoError(t, err)
	assert.Subset(t, walkedNames, expectedNames)
}

func TestCachedWalkByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("WalkByQuery")
	err := store.UpsertMany(cachedStoreCtx, testObjects)
	require.NoError(t, err)

	query2 := getCachedMatchFieldQuery("Test Name", "Test WalkByQuery 2")
	query4 := getCachedMatchFieldQuery("Test Key", "TestWalkByQuery4")
	query := getCachedDisjunctionQuery(query2, query4)

	walkedNames := make([]string, 0, len(testObjects))
	walkedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedNames = append(walkedNames, obj.Name)
		walkedObjects = append(walkedObjects, obj)
		return nil
	}
	err = store.WalkByQuery(cachedStoreCtx, query, walkFn)
	require.NoError(t, err)

	expectedNames := []string{
		"Test WalkByQuery 2",
		"Test WalkByQuery 4",
	}
	expectedObjects := []*storage.TestSingleKeyStruct{
		testObjects[1],
		testObjects[3],
	}
	assert.ElementsMatch(t, expectedNames, walkedNames)
	protoassert.ElementsMatch(t, expectedObjects, walkedObjects)
}

func TestCachedWalkByQueryContextCancelation(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("WalkByQuery")
	err := store.UpsertMany(cachedStoreCtx, testObjects)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(cachedStoreCtx)
	cancel()
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		return nil
	}
	err = store.WalkByQuery(ctx, nil, walkFn)

	assert.ErrorIs(t, err, context.Canceled)

	q := getCachedMatchFieldQuery("Test Name", "Test WalkByQuery 2")
	err = store.WalkByQuery(ctx, q, walkFn)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestCachedGetIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("GetIDs")
	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	expectedIDs := make([]string, 0, len(testObjects))
	for _, obj := range testObjects {
		expectedIDs = append(expectedIDs, pkGetterForCache(obj))
	}

	fetchedIDs, err := store.GetIDs(cachedStoreCtx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedIDs, expectedIDs)
}

func TestCachedGet(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	key := "TestGet"
	name := "Test Get"
	testObject := newCachedTestSingleKeyStruct(key, name, int64(15))

	missingKey := "TestGetMissing"

	assert.NoError(t, store.Upsert(cachedStoreCtx, testObject))

	// Object with ID "TestGet" is in DB
	obj, found, err := store.Get(cachedStoreCtx, key)
	protoassert.Equal(t, testObject, obj)
	assert.True(t, found)
	assert.NoError(t, err)

	// Object with ID "TestGetMissing" is NOT in DB
	missingObj, missingFound, missingErr := store.Get(cachedStoreCtx, missingKey)
	assert.Nil(t, missingObj)
	assert.False(t, missingFound)
	assert.NoError(t, missingErr)
}

func TestCachedGetMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)

	missingKey := "TestGetManyMissing"
	testObjects := sampleCachedTestSingleKeyStructArray("GetMany")
	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	identifiersToFetch := make([]string, 0, len(testObjects)+1)
	expectedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))
	identifiersToFetch = append(identifiersToFetch, missingKey)
	for ix, obj := range testObjects {
		if ix%2 == 1 {
			continue
		}
		identifiersToFetch = append(identifiersToFetch, pkGetterForCache(obj))
		expectedObjects = append(expectedObjects, obj)
	}

	fetchedObjects, missingIndices, err := store.GetMany(cachedStoreCtx, identifiersToFetch)
	assert.NoError(t, err)
	protoassert.ElementsMatch(t, fetchedObjects, expectedObjects)
	assert.Equal(t, []int{0}, missingIndices)
}

func TestCachedGetByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)

	testObjects := sampleCachedTestSingleKeyStructArray("GetByQuery")
	query2 := getCachedMatchFieldQuery("Test Name", "Test GetByQuery 2")
	query4 := getCachedMatchFieldQuery("Test Key", "TestGetByQuery4")
	query := getCachedDisjunctionQuery(query2, query4)

	objectsBefore, errBefore := store.GetByQuery(cachedStoreCtx, query)
	assert.NoError(t, errBefore)
	assert.Empty(t, objectsBefore)

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	objectsAfter, errAfter := store.GetByQuery(cachedStoreCtx, query)
	assert.NoError(t, errAfter)
	expectedObjectsAfter := []*storage.TestSingleKeyStruct{
		testObjects[1],
		testObjects[3],
	}
	protoassert.ElementsMatch(t, objectsAfter, expectedObjectsAfter)
}

func TestCachedDeleteByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)

	testObjects := sampleCachedTestSingleKeyStructArray("DeleteByQuery")
	query2 := getCachedMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getCachedMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getCachedDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(cachedStoreCtx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	_, deleteFromEmptyErr := store.DeleteByQuery(cachedStoreCtx, query)
	assert.NoError(t, deleteFromEmptyErr)

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		protoassert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	_, deleteFromPopulatedErr := store.DeleteByQuery(cachedStoreCtx, query)
	assert.NoError(t, deleteFromPopulatedErr)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			protoassert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

func TestCachedDeleteByQueryReturningIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)

	testObjects := sampleCachedTestSingleKeyStructArray("DeleteByQuery")
	query2 := getCachedMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getCachedMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getCachedDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(cachedStoreCtx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	deletedIDsFromEmpty, deleteFromEmptyErr := store.DeleteByQuery(cachedStoreCtx, query)
	assert.NoError(t, deleteFromEmptyErr)
	assert.Empty(t, deletedIDsFromEmpty)

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		protoassert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	deletedIDsFromPopulated, deleteFromPopulatedErr := store.DeleteByQuery(cachedStoreCtx, query)
	assert.NoError(t, deleteFromPopulatedErr)
	expectedIDs := []string{pkGetterForCache(testObjects[1]), pkGetterForCache(testObjects[3])}
	assert.ElementsMatch(t, deletedIDsFromPopulated, expectedIDs)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			protoassert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

// region Helper Functions

func newCachedStore(testDB *pgtest.TestPostgres) Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct] {
	return NewGenericStoreWithCache[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetterForCache,
		insertIntoTestSingleKeyStructsWithCache,
		copyFromTestSingleKeyStructsWithCache,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		globallyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	)
}

func newCachedTestSingleKeyStruct(key string, name string, intVal int64) *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:   key,
		Name:  name,
		Int64: intVal,
	}
}

func sampleCachedTestSingleKeyStructArray(pattern string) []*storage.TestSingleKeyStruct {
	output := make([]*storage.TestSingleKeyStruct, 0, 5)
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("Test%s%d", pattern, i)
		name := fmt.Sprintf("Test %s %d", pattern, i)
		output = append(output, &storage.TestSingleKeyStruct{Key: key, Name: name, Int64: int64(i)})
	}
	return output
}

func getCachedMatchFieldQuery(fieldName string, value string) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field:     fieldName,
						Value:     value,
						Highlight: false,
					},
				},
			},
		},
	}
}

func getCachedDisjunctionQuery(q1 *v1.Query, q2 *v1.Query) *v1.Query {
	if q1 == nil && q2 == nil {
		return nil
	}
	if q1 == nil {
		return q2
	}
	if q2 == nil {
		return q1
	}
	return &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{q1, q2},
			},
		},
	}
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func pkGetterForCache(obj *storage.TestSingleKeyStruct) string {
	return obj.GetKey()
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func insertIntoTestSingleKeyStructsWithCache(batch *pgx.Batch, obj *storage.TestSingleKeyStruct) error {
	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetKey(),
		obj.GetName(),
		obj.GetStringSlice(),
		obj.GetBool(),
		obj.GetUint64(),
		obj.GetInt64(),
		obj.GetFloat(),
		pgutils.EmptyOrMap(obj.GetLabels()),
		protocompat.NilOrTime(obj.GetTimestamp()),
		obj.GetEnum(),
		obj.GetEnums(),
		serialized,
	}

	finalStr := "INSERT INTO test_single_key_structs (Key, Name, StringSlice, Bool, Uint64, Int64, Float, Labels, Timestamp, Enum, Enums, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT(Key) DO UPDATE SET Key = EXCLUDED.Key, Name = EXCLUDED.Name, StringSlice = EXCLUDED.StringSlice, Bool = EXCLUDED.Bool, Uint64 = EXCLUDED.Uint64, Int64 = EXCLUDED.Int64, Float = EXCLUDED.Float, Labels = EXCLUDED.Labels, Timestamp = EXCLUDED.Timestamp, Enum = EXCLUDED.Enum, Enums = EXCLUDED.Enums, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func copyFromTestSingleKeyStructsWithCache(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*storage.TestSingleKeyStruct) error {
	batchSize := MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy, so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"key",
		"name",
		"stringslice",
		"bool",
		"uint64",
		"int64",
		"float",
		"labels",
		"timestamp",
		"enum",
		"enums",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetKey(),
			obj.GetName(),
			obj.GetStringSlice(),
			obj.GetBool(),
			obj.GetUint64(),
			obj.GetInt64(),
			obj.GetFloat(),
			pgutils.EmptyOrMap(obj.GetLabels()),
			protocompat.NilOrTime(obj.GetTimestamp()),
			obj.GetEnum(),
			obj.GetEnums(),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetKey())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and values for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"test_single_key_structs"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion
