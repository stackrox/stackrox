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
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/uuid"
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
		nil,
		nil,
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
		nil,
		nil,
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

	// Now test with scoped context
	// Test with a valid scoped context using the test schema's own category
	// Note: TestSingleKeyStructsSchema uses v1.SearchCategory(100) so we'll create
	// a scoped context that would be valid for schemas that reference this category
	validScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{"TestCount1", "TestCount2"}, // Scope to specific object IDs
		Level: v1.SearchCategory(100),               // Use the TestSingleKeyStruct's own category
	})

	fourthCount, err4 := store.Count(validScopedCtx, nil)
	assert.Equal(t, 2, fourthCount)
	assert.NoError(t, err4)

	// Count with valid scope and valid query, query restricts more than scope
	countQuery := getCachedMatchFieldQuery("Test Name", "Test Count 1")
	count5, err5 := store.Count(validScopedCtx, countQuery)
	assert.Equal(t, 1, count5)
	assert.NoError(t, err5)
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
		walkedNames = append(walkedNames, obj.GetName())
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
		walkedNames = append(walkedNames, obj.GetName())
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
		walkedNames = append(walkedNames, obj.GetName())
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

func TestCachedWalkByQueryScopedContext(t *testing.T) {
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
		walkedNames = append(walkedNames, obj.GetName())
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	// First use a scope that doesn't match to so we can simply re-use the function and objects
	noMatchScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{"TestCount1", "TestCount2"}, // Scope to specific object IDs
		Level: v1.SearchCategory(100),               // Use the TestSingleKeyStruct's own category
	})
	err = store.WalkByQuery(noMatchScopedCtx, query, walkFn)
	require.NoError(t, err)
	assert.Equal(t, 0, len(walkedNames))
	assert.Equal(t, 0, len(walkedObjects))

	matchScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{"TestWalkByQuery2"}, // Scope to specific object IDs
		Level: v1.SearchCategory(100),       // Use the TestSingleKeyStruct's own category
	})
	err = store.WalkByQuery(matchScopedCtx, query, walkFn)
	require.NoError(t, err)
	expectedNames := []string{
		"Test WalkByQuery 2",
	}
	expectedObjects := []*storage.TestSingleKeyStruct{
		testObjects[1],
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

	deleteFromEmptyErr := store.DeleteByQuery(cachedStoreCtx, query)
	assert.NoError(t, deleteFromEmptyErr)

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		protoassert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	deleteFromPopulatedErr := store.DeleteByQuery(cachedStoreCtx, query)
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

	deletedIDsFromEmpty, deleteFromEmptyErr := store.DeleteByQueryWithIDs(cachedStoreCtx, query)
	assert.NoError(t, deleteFromEmptyErr)
	assert.Empty(t, deletedIDsFromEmpty)

	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(cachedStoreCtx, pkGetterForCache(obj))
		protoassert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	deletedIDsFromPopulated, deleteFromPopulatedErr := store.DeleteByQueryWithIDs(cachedStoreCtx, query)
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

// Scoped Context Tests

func TestCachedCountWithInvalidScopedContext(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	// Setup test data
	testObjects := sampleCachedTestSingleKeyStructArray("ScopedCount")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	// Test 1: Count with invalid image scope for TestSingleKeyStruct schema and nil query
	// The search framework should filter out invalid scope queries, leaving a nil query which returns no results
	imageScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{"fake-image-id"},
		Level: v1.SearchCategory_IMAGES,
	})

	count, err := store.Count(imageScopedCtx, nil)
	assert.NoError(t, err)
	// Since the scope is invalid and gets filtered out, and the original query was nil, the result is no results
	assert.Equal(t, 0, count)

	// Test 2: Count with query and invalid scope should behave normally since scope is filtered out but query remains valid
	query := getCachedMatchFieldQuery("Test Name", "Test ScopedCount 1")
	count, err = store.Count(imageScopedCtx, query)
	assert.NoError(t, err)
	// Should fallback to underlying store due to query, scope gets filtered out but query is still valid
	assert.Equal(t, 1, count) // Should find the matching object

	// Test 3: Count with nil query should use cache when no scope
	count, err = store.Count(cachedStoreCtx, nil)
	assert.NoError(t, err)
	assert.Equal(t, len(testObjects), count)

	// Test 4: Count with empty query should use cache when no scope
	count, err = store.Count(cachedStoreCtx, search.EmptyQuery())
	assert.NoError(t, err)
	assert.Equal(t, len(testObjects), count)
}

func TestCachedGetByQueryFnWithInvalidScopedContext(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("ScopedGetByQueryFn")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	// Test 1: GetByQueryFn with invalid image scope for TestSingleKeyStruct schema
	// The search framework should filter out invalid scope queries
	imageScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{"fake-image-id"},
		Level: v1.SearchCategory_IMAGES,
	})

	var walkedObjects []*storage.TestSingleKeyStruct
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	query := getCachedMatchFieldQuery("Test Name", "Test ScopedGetByQueryFn 2")
	err := store.GetByQueryFn(imageScopedCtx, query, walkFn)
	assert.NoError(t, err)
	// Since scope is invalid and filtered out, should find the matching object
	assert.Len(t, walkedObjects, 1)
	assert.Equal(t, "Test ScopedGetByQueryFn 2", walkedObjects[0].GetName())

	// Test 2: GetByQueryFn with nil query and invalid scope results in nil query after filtering
	// Since scope is invalid and gets filtered out, and the original query was nil, the result is no results
	walkedObjects = nil
	err = store.GetByQueryFn(imageScopedCtx, nil, walkFn)
	assert.NoError(t, err)
	assert.Empty(t, walkedObjects) // Should find no objects since filtered query becomes nil

	// Test 3: GetByQueryFn without scope should work normally (baseline test)
	walkedObjects = nil
	err = store.GetByQueryFn(cachedStoreCtx, query, walkFn)
	assert.NoError(t, err)
	assert.Len(t, walkedObjects, 1)
	assert.Equal(t, "Test ScopedGetByQueryFn 2", walkedObjects[0].GetName())

	// Test 4: GetByQueryFn with empty query and no scope should use cache
	walkedObjects = nil
	err = store.GetByQueryFn(cachedStoreCtx, search.EmptyQuery(), walkFn)
	assert.NoError(t, err)
	assert.Len(t, walkedObjects, len(testObjects))
}

func TestCachedWalkByQueryWithInvalidScopedContext(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("ScopedWalkByQuery")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	// Test 1: WalkByQuery with invalid deployment scope for TestSingleKeyStruct schema
	// The search framework should filter out invalid scope queries
	deploymentScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{uuid.NewV4().String()},
		Level: v1.SearchCategory_DEPLOYMENTS,
	})

	var walkedNames []string
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedNames = append(walkedNames, obj.GetName())
		return nil
	}

	query := getCachedMatchFieldQuery("Test Name", "Test ScopedWalkByQuery 3")
	err := store.WalkByQuery(deploymentScopedCtx, query, walkFn)
	assert.NoError(t, err)
	// Since scope is invalid and filtered out, should find the matching object
	assert.Len(t, walkedNames, 1)
	assert.Contains(t, walkedNames, "Test ScopedWalkByQuery 3")

	// Test 2: WalkByQuery without scope should work normally (baseline test)
	walkedNames = nil
	err = store.WalkByQuery(cachedStoreCtx, query, walkFn)
	assert.NoError(t, err)
	assert.Len(t, walkedNames, 1)
	assert.Contains(t, walkedNames, "Test ScopedWalkByQuery 3")
}

func TestCachedDeleteByQueryWithInvalidScopedContext(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("ScopedDeleteByQuery")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	// Test 1: DeleteByQuery with invalid namespace scope for TestSingleKeyStruct schema
	// The search framework should filter out invalid scope queries, making this behave normally
	namespaceScopedCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{uuid.NewV4().String()},
		Level: v1.SearchCategory_NAMESPACES,
	})

	query := getCachedMatchFieldQuery("Test Name", "Test ScopedDeleteByQuery 1")
	err := store.DeleteByQuery(namespaceScopedCtx, query)
	assert.NoError(t, err)

	// Verify object was actually deleted (scope was ignored, normal deletion occurred)
	obj, found, err := store.Get(cachedStoreCtx, "TestScopedDeleteByQuery1")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, obj)

	// Test 2: DeleteByQuery without scope should work normally (baseline test with different object)
	query2 := getCachedMatchFieldQuery("Test Name", "Test ScopedDeleteByQuery 2")
	deletedIDs, err := store.DeleteByQueryWithIDs(cachedStoreCtx, query2)
	assert.NoError(t, err)
	assert.Len(t, deletedIDs, 1)
	assert.Contains(t, deletedIDs, "TestScopedDeleteByQuery2")

	// Verify second object was actually deleted
	obj, found, err = store.Get(cachedStoreCtx, "TestScopedDeleteByQuery2")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, obj)
}

func TestCachedStoreMultipleInvalidScopedLevels(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("MultiScope")
	require.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	// Create nested invalid scoped context for TestSingleKeyStruct schema: cluster -> namespace -> deployment
	// All these scopes should be filtered out by the search framework
	clusterID := uuid.NewV4().String()
	namespaceID := uuid.NewV4().String()
	deploymentID := uuid.NewV4().String()

	clusterCtx := scoped.Context(cachedStoreCtx, scoped.Scope{
		IDs:   []string{clusterID},
		Level: v1.SearchCategory_CLUSTERS,
	})

	namespaceCtx := scoped.Context(clusterCtx, scoped.Scope{
		IDs:   []string{namespaceID},
		Level: v1.SearchCategory_NAMESPACES,
	})

	deploymentCtx := scoped.Context(namespaceCtx, scoped.Scope{
		IDs:   []string{deploymentID},
		Level: v1.SearchCategory_DEPLOYMENTS,
	})

	// Test Count with nested invalid scopes and nil query results in nil query after filtering
	count, err := store.Count(deploymentCtx, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // Should find no objects since filtered query becomes nil

	// Test GetByQuery with nested invalid scopes - should behave normally
	query := getCachedMatchFieldQuery("Test Name", "Test MultiScope 2")
	results, err := store.GetByQuery(deploymentCtx, query)
	assert.NoError(t, err)
	assert.Len(t, results, 1) // Should find the matching object since scopes are filtered out

	// Test that the same operations work without scope (baseline tests)
	count, err = store.Count(cachedStoreCtx, nil)
	assert.NoError(t, err)
	assert.Equal(t, len(testObjects), count)

	results, err = store.GetByQuery(cachedStoreCtx, query)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCachedGetAllFromCache(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newCachedStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleCachedTestSingleKeyStructArray("GetAllFromCache")
	assert.NoError(t, store.UpsertMany(cachedStoreCtx, testObjects))

	protoassert.ElementsMatch(t, testObjects, store.GetAllFromCache())
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
		nil,
		nil,
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
