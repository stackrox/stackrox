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
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestNewStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, newStore(testDB))
}

func TestNewGenericStore(t *testing.T) {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, NewGenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetter,
		insertIntoTestSingleKeyStructs,
		copyFromTestSingleKeyStructs,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		GloballyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	))
}

func TestUpsert(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	key := "TestUpsert"
	name := "Test Upsert"
	testObject := newTestSingleKeyStruct(key, name, int64(1))

	objBefore, foundBefore, errBefore := store.Get(ctx, key)
	assert.Nil(t, objBefore)
	assert.False(t, foundBefore)
	assert.NoError(t, errBefore)

	assert.NoError(t, store.Upsert(ctx, testObject))

	objAfter, foundAfter, errAfter := store.Get(ctx, key)
	assert.Equal(t, testObject, objAfter)
	assert.True(t, foundAfter)
	assert.NoError(t, errAfter)
}

func TestUpsertMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("UpsertMany")

	for _, obj := range testObjects {
		objBefore, foundBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Nil(t, objBefore)
		assert.False(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	for _, obj := range testObjects {
		objAfter, foundAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objAfter)
		assert.True(t, foundAfter)
		assert.NoError(t, errAfter)
	}
}

func TestDelete(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	key := "TestDelete"
	name := "Test Delete"
	testObject := newTestSingleKeyStruct(key, name, int64(1))
	require.NoError(t, store.Upsert(ctx, testObject))

	objBefore, foundBefore, errBefore := store.Get(ctx, key)
	require.Equal(t, testObject, objBefore)
	require.True(t, foundBefore)
	require.NoError(t, errBefore)

	assert.NoError(t, store.Delete(ctx, key))

	objAfter, foundAfter, errAfter := store.Get(ctx, key)
	require.Nil(t, objAfter)
	require.False(t, foundAfter)
	require.NoError(t, errAfter)

	missingKey := "TestDeleteMissingKey"

	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(ctx, missingKey)
	require.Nil(t, missingObjBefore)
	require.False(t, missingFoundBefore)
	require.NoError(t, missingErrBefore)

	assert.NoError(t, store.Delete(ctx, missingKey))

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(ctx, missingKey)
	require.Nil(t, missingObjAfter)
	require.False(t, missingFoundAfter)
	require.NoError(t, missingErrAfter)

}

func TestDeleteMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	objectBatch := sampleTestSingleKeyStructArray("DeleteMany")
	require.NoError(t, store.UpsertMany(ctx, objectBatch))

	identifiersToRemove := make([]string, 0, len(objectBatch)+1)
	for _, obj := range objectBatch {
		key := pkGetter(obj)
		identifiersToRemove = append(identifiersToRemove, key)
		// ensure object is in DB before call to remove
		objBefore, foundBefore, errBefore := store.Get(ctx, key)
		assert.Equal(t, obj, objBefore)
		assert.True(t, foundBefore)
		assert.NoError(t, errBefore)
	}

	missingKey := "TestDeleteManyMissingKey"
	identifiersToRemove = append(identifiersToRemove, missingKey)
	missingObjBefore, missingFoundBefore, missingErrBefore := store.Get(ctx, missingKey)
	assert.Nil(t, missingObjBefore)
	assert.False(t, missingFoundBefore)
	assert.NoError(t, missingErrBefore)

	assert.NoError(t, store.DeleteMany(ctx, identifiersToRemove))

	for _, obj := range objectBatch {
		key := pkGetter(obj)
		// ensure object is NOT in DB after call to remove
		objAfter, foundAfter, errAfter := store.Get(ctx, key)
		assert.Nil(t, objAfter)
		assert.False(t, foundAfter)
		assert.NoError(t, errAfter)
	}

	missingObjAfter, missingFoundAfter, missingErrAfter := store.Get(ctx, missingKey)
	assert.Nil(t, missingObjAfter)
	assert.False(t, missingFoundAfter)
	assert.NoError(t, missingErrAfter)
}

func TestExists(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	key := "TestExists"
	name := "Test Exists"
	testObject := newTestSingleKeyStruct(key, name, int64(9))

	require.NoError(t, store.Upsert(ctx, testObject))

	missingKey := "TestExistsMissingKey"

	foundExisting, errExisting := store.Exists(ctx, key)
	assert.True(t, foundExisting)
	assert.NoError(t, errExisting)

	foundMissing, errMissing := store.Exists(ctx, missingKey)
	assert.False(t, foundMissing)
	assert.NoError(t, errMissing)
}

func TestCount(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	firstCount, err1 := store.Count(ctx)
	assert.Equal(t, 0, firstCount)
	assert.NoError(t, err1)

	testObject1 := newTestSingleKeyStruct("TestCount", "Test Count", int64(256))
	assert.NoError(t, store.Upsert(ctx, testObject1))

	secondCount, err2 := store.Count(ctx)
	assert.Equal(t, 1, secondCount)
	assert.NoError(t, err2)

	supplementaryObjects := sampleTestSingleKeyStructArray("Count")
	assert.NoError(t, store.UpsertMany(ctx, supplementaryObjects))

	thirdCount, err3 := store.Count(ctx)
	assert.Equal(t, 1+len(supplementaryObjects), thirdCount)
	assert.NoError(t, err3)
}

func TestWalk(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("Walk")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

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

	assert.NoError(t, store.Walk(ctx, walkFn))

	assert.ElementsMatch(t, walkedNames, injectedNames)
	assert.ElementsMatch(t, testObjects, walkedObjects)
}

func TestWalkByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("WalkByQuery")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	query := search.NewQueryBuilder().AddExactMatches(search.TestName, testObjects[0].GetName(), testObjects[1].GetName()).ProtoQuery()
	expectedObjects := testObjects[:2]
	var walkedObjects []*storage.TestSingleKeyStruct
	walkFn := func(obj *storage.TestSingleKeyStruct) error {
		walkedObjects = append(walkedObjects, obj)
		return nil
	}

	assert.NoError(t, store.WalkByQuery(ctx, query, walkFn))
	assert.ElementsMatch(t, expectedObjects, walkedObjects)
}

func TestGetAll(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("GetAll")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	fetchedObjects, err := store.GetAll(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedObjects, testObjects)
}

func TestGetIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	require.NotNil(t, store)

	testObjects := sampleTestSingleKeyStructArray("GetIDs")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	expectedIDs := make([]string, 0, len(testObjects))
	for _, obj := range testObjects {
		expectedIDs = append(expectedIDs, pkGetter(obj))
	}

	fetchedIDs, err := store.GetIDs(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedIDs, expectedIDs)
}

func TestGet(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)
	key := "TestGet"
	name := "Test Get"
	testObject := newTestSingleKeyStruct(key, name, int64(15))

	missingKey := "TestGetMissing"

	assert.NoError(t, store.Upsert(ctx, testObject))

	// Object with ID "TestGet" is in DB
	obj, found, err := store.Get(ctx, key)
	assert.Equal(t, testObject, obj)
	assert.True(t, found)
	assert.NoError(t, err)

	// Object with ID "TestGetMissing" is NOT in DB
	missingObj, missingFound, missingErr := store.Get(ctx, missingKey)
	assert.Nil(t, missingObj)
	assert.False(t, missingFound)
	assert.NoError(t, missingErr)
}

func TestGetMany(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	missingKey := "TestGetManyMissing"
	testObjects := sampleTestSingleKeyStructArray("GetMany")
	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	identifiersToFetch := make([]string, 0, len(testObjects)+1)
	expectedObjects := make([]*storage.TestSingleKeyStruct, 0, len(testObjects))
	identifiersToFetch = append(identifiersToFetch, missingKey)
	for ix, obj := range testObjects {
		if ix%2 == 1 {
			continue
		}
		identifiersToFetch = append(identifiersToFetch, pkGetter(obj))
		expectedObjects = append(expectedObjects, obj)
	}

	fetchedObjects, missingIndices, err := store.GetMany(ctx, identifiersToFetch)
	assert.NoError(t, err)
	assert.ElementsMatch(t, fetchedObjects, expectedObjects)
	assert.Equal(t, []int{0}, missingIndices)
}

func TestGetByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("GetByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test GetByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestGetByQuery4")
	query := getDisjunctionQuery(query2, query4)

	objectsBefore, errBefore := store.GetByQuery(ctx, query)
	assert.NoError(t, errBefore)
	assert.Empty(t, objectsBefore)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))

	objectsAfter, errAfter := store.GetByQuery(ctx, query)
	assert.NoError(t, errAfter)
	expectedObjectsAfter := []*storage.TestSingleKeyStruct{
		testObjects[1],
		testObjects[3],
	}
	assert.ElementsMatch(t, objectsAfter, expectedObjectsAfter)
}

func TestDeleteByQuery(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("DeleteByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(ctx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	_, deleteFromEmptyErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromEmptyErr)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	_, deleteFromPopulatedErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromPopulatedErr)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			assert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

func TestDeleteByQueryReturningIDs(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := newStore(testDB)

	testObjects := sampleTestSingleKeyStructArray("DeleteByQuery")
	query2 := getMatchFieldQuery("Test Name", "Test DeleteByQuery 2")
	query4 := getMatchFieldQuery("Test Key", "TestDeleteByQuery4")
	query := getDisjunctionQuery(query2, query4)

	queriedObjectsFromEmpty, errQueryFromEmpty := store.GetByQuery(ctx, query)
	assert.NoError(t, errQueryFromEmpty)
	assert.Empty(t, queriedObjectsFromEmpty)

	deletedIDsFromEmpty, deleteFromEmptyErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromEmptyErr)
	assert.Empty(t, deletedIDsFromEmpty)

	assert.NoError(t, store.UpsertMany(ctx, testObjects))
	for _, obj := range testObjects {
		objBefore, fetchedBefore, errBefore := store.Get(ctx, pkGetter(obj))
		assert.Equal(t, obj, objBefore)
		assert.True(t, fetchedBefore)
		assert.NoError(t, errBefore)
	}

	deletedIDsFromPopulated, deleteFromPopulatedErr := store.DeleteByQuery(ctx, query)
	assert.NoError(t, deleteFromPopulatedErr)
	expectedIDs := []string{pkGetter(testObjects[1]), pkGetter(testObjects[3])}
	assert.ElementsMatch(t, deletedIDsFromPopulated, expectedIDs)

	for idx, obj := range testObjects {
		objAfter, fetchedAfter, errAfter := store.Get(ctx, pkGetter(obj))
		assert.NoError(t, errAfter)
		if idx == 1 || idx == 3 {
			assert.Nil(t, objAfter)
			assert.False(t, fetchedAfter)
		} else {
			assert.Equal(t, obj, objAfter)
			assert.True(t, fetchedAfter)
		}
	}
}

// region Helper Functions

func newStore(testDB *pgtest.TestPostgres) Store[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct] {
	return NewGenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetter,
		insertIntoTestSingleKeyStructs,
		copyFromTestSingleKeyStructs,
		doNothingDurationTimeSetter,
		doNothingDurationTimeSetter,
		GloballyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	)
}

func newTestSingleKeyStruct(key string, name string, intVal int64) *storage.TestSingleKeyStruct {
	return &storage.TestSingleKeyStruct{
		Key:   key,
		Name:  name,
		Int64: intVal,
	}
}

func sampleTestSingleKeyStructArray(pattern string) []*storage.TestSingleKeyStruct {
	output := make([]*storage.TestSingleKeyStruct, 0, 5)
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("Test%s%d", pattern, i)
		name := fmt.Sprintf("Test %s %d", pattern, i)
		output = append(output, &storage.TestSingleKeyStruct{Key: key, Name: name, Int64: int64(i)})
	}
	return output
}

func getMatchFieldQuery(fieldName string, value string) *v1.Query {
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

func getDisjunctionQuery(q1 *v1.Query, q2 *v1.Query) *v1.Query {
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
func pkGetter(obj *storage.TestSingleKeyStruct) string {
	return obj.GetKey()
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func insertIntoTestSingleKeyStructs(batch *pgx.Batch, obj *storage.TestSingleKeyStruct) error {

	serialized, marshalErr := obj.Marshal()
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
		pgutils.NilOrTime(obj.GetTimestamp()),
		obj.GetEnum(),
		obj.GetEnums(),
		serialized,
	}

	finalStr := "INSERT INTO test_single_key_structs (Key, Name, StringSlice, Bool, Uint64, Int64, Float, Labels, Timestamp, Enum, Enums, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT(Key) DO UPDATE SET Key = EXCLUDED.Key, Name = EXCLUDED.Name, StringSlice = EXCLUDED.StringSlice, Bool = EXCLUDED.Bool, Uint64 = EXCLUDED.Uint64, Int64 = EXCLUDED.Int64, Float = EXCLUDED.Float, Labels = EXCLUDED.Labels, Timestamp = EXCLUDED.Timestamp, Enum = EXCLUDED.Enum, Enums = EXCLUDED.Enums, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// copied from tools/generate-helpers/pg-table-bindings/test/postgres/store.go
func copyFromTestSingleKeyStructs(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*storage.TestSingleKeyStruct) error {
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

		serialized, marshalErr := obj.Marshal()
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
			pgutils.NilOrTime(obj.GetTimestamp()),
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
