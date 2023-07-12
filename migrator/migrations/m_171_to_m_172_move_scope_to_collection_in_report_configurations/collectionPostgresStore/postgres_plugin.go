package postgres

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

// This file is a partial copy of central/resourcecollection/datastore/store/postgres/store.go
// in the state it had when the migration was written.
// Only the relevant functions (Upsert, UpsertMany, DeleteMany, Get and Walk) are kept.
// The kept functions are stripped from the scoped access control checks.

const (
	baseTable = "collections"

	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000

	cursorBatchSize = 50
	deleteBatchSize = 5000
)

var (
	log    = logging.LoggerForModule()
	schema = pkgSchema.CollectionsSchema
)

// Store is the interface to interact with the storage for storage.ResourceCollection
type Store interface {
	Upsert(ctx context.Context, obj *storage.ResourceCollection) error
	UpsertMany(ctx context.Context, objs []*storage.ResourceCollection) error
	DeleteMany(ctx context.Context, identifiers []string) error

	Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error)
	Walk(ctx context.Context, fn func(obj *storage.ResourceCollection) error) error
}

type storeImpl struct {
	db    postgres.DB
	mutex sync.Mutex
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

//// Helper functions

func insertIntoCollections(ctx context.Context, batch *pgx.Batch, obj *storage.ResourceCollection) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		obj.GetCreatedBy().GetName(),
		obj.GetUpdatedBy().GetName(),
		serialized,
	}

	finalStr := "INSERT INTO collections (Id, Name, CreatedBy_Name, UpdatedBy_Name, serialized) VALUES($1, $2, $3, $4, $5) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, CreatedBy_Name = EXCLUDED.CreatedBy_Name, UpdatedBy_Name = EXCLUDED.UpdatedBy_Name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	var query string

	for childIndex, child := range obj.GetEmbeddedCollections() {
		if err := insertIntoCollectionsEmbeddedCollections(ctx, batch, child, obj.GetId(), childIndex); err != nil {
			return err
		}
	}

	query = "delete from collections_embedded_collections where collections_Id = $1 AND idx >= $2"
	batch.Queue(query, obj.GetId(), len(obj.GetEmbeddedCollections()))
	return nil
}

func insertIntoCollectionsEmbeddedCollections(_ context.Context, batch *pgx.Batch, obj *storage.ResourceCollection_EmbeddedResourceCollection, collectionsID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		collectionsID,
		idx,
		obj.GetId(),
	}

	finalStr := "INSERT INTO collections_embedded_collections (collections_Id, idx, Id) VALUES($1, $2, $3) ON CONFLICT(collections_Id, idx) DO UPDATE SET collections_Id = EXCLUDED.collections_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id"
	batch.Queue(finalStr, values...)

	return nil
}

func (s *storeImpl) copyFromCollections(ctx context.Context, tx *postgres.Tx, objs ...*storage.ResourceCollection) error {

	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"name",

		"createdby_name",

		"updatedby_name",

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

			obj.GetId(),

			obj.GetName(),

			obj.GetCreatedBy().GetName(),

			obj.GetUpdatedBy().GetName(),

			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"collections"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	for idx, obj := range objs {
		_ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.

		if err = s.copyFromCollectionsEmbeddedCollections(ctx, tx, obj.GetId(), obj.GetEmbeddedCollections()...); err != nil {
			return err
		}
	}

	return err
}

func (s *storeImpl) copyFromCollectionsEmbeddedCollections(ctx context.Context, tx *postgres.Tx, collectionsID string, objs ...*storage.ResourceCollection_EmbeddedResourceCollection) error {

	inputRows := [][]interface{}{}

	var err error

	copyCols := []string{

		"collections_id",

		"idx",

		"id",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		inputRows = append(inputRows, []interface{}{

			collectionsID,

			idx,

			obj.GetId(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"collections_embedded_collections"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.ResourceCollection) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "ResourceCollection")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromCollections(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.ResourceCollection) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "ResourceCollection")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoCollections(ctx, batch, obj); err != nil {
			return err
		}
		batchResults := conn.SendBatch(ctx, batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err := batchResults.Exec()
			result = multierror.Append(result, err)
		}
		if err := batchResults.Close(); err != nil {
			return err
		}
		if err := result.ErrorOrNil(); err != nil {
			return err
		}
	}
	return nil
}

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.ResourceCollection) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

//// Helper functions - END

//// Interface functions

// UpsertMany saves the state of multiple objects in the storage.
func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.ResourceCollection) error {
	return pgutils.Retry(func() error {
		// Lock since copyFrom requires a delete first before being executed.  If multiple processes are updating
		// same subset of rows, both deletes could occur before the copyFrom resulting in unique constraint
		// violations
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if len(objs) < batchAfter {
			return s.upsert(ctx, objs...)
		}
		return s.copyFrom(ctx, objs...)
	})
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *storeImpl) DeleteMany(ctx context.Context, identifiers []string) error {
	var sacQueryFilter *v1.Query

	// Batch the deletes
	localBatchSize := deleteBatchSize
	numRecordsToDelete := len(identifiers)
	for {
		if len(identifiers) == 0 {
			break
		}

		if len(identifiers) < localBatchSize {
			localBatchSize = len(identifiers)
		}

		identifierBatch := identifiers[:localBatchSize]
		q := search.ConjunctionQuery(
			sacQueryFilter,
			search.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery(),
		)

		if err := pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db); err != nil {
			err = errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(identifiers), numRecordsToDelete)
			log.Error(err)
			return err
		}

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	return nil
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error) {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	data, err := pgSearch.RunGetQueryForSchema[storage.ResourceCollection](ctx, schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}

// Walk iterates over all of the objects in the store and applies the closure.
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.ResourceCollection) error) error {
	var sacQueryFilter *v1.Query
	fetcher, closer, err := pgSearch.RunCursorQueryForSchema[storage.ResourceCollection](ctx, schema, sacQueryFilter, s.db)
	if err != nil {
		return err
	}
	defer closer()
	for {
		rows, err := fetcher(cursorBatchSize)
		if err != nil {
			return pgutils.ErrNilIfNoRows(err)
		}
		for _, data := range rows {
			if err := fn(data); err != nil {
				return err
			}
		}
		if len(rows) != cursorBatchSize {
			break
		}
	}
	return nil
}
