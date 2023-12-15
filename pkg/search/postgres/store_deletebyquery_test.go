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
	"github.com/stretchr/testify/assert"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

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

func newStore(testDB *pgtest.TestPostgres) *GenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct] {
	return NewGenericStore[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](
		testDB.DB,
		pkgSchema.TestSingleKeyStructsSchema,
		pkGetter,
		insertIntoTestSingleKeyStructs,
		copyFromTestSingleKeyStructs,
		nil,
		nil,
		GloballyScopedUpsertChecker[storage.TestSingleKeyStruct, *storage.TestSingleKeyStruct](resources.Namespace),
		resources.Namespace,
	)
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
