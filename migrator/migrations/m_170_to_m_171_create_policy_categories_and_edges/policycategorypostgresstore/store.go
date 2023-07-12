package postgres

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	baseTable  = "policy_categories"
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
	schema = frozenSchema.PolicyCategoriesSchema
)

// Store is the interface to interact with the storage for storage.PolicyCategory
type Store interface {
	Count(ctx context.Context) (int, error)
	Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error

	Upsert(ctx context.Context, obj *storage.PolicyCategory) error
	UpsertMany(ctx context.Context, objs []*storage.PolicyCategory) error

	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategory, error)
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

// GetByQuery returns the objects from the store matching the query.
func (s *storeImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategory, error) {
	var sacQueryFilter *v1.Query

	q := search.ConjunctionQuery(
		sacQueryFilter,
		query,
	)

	rows, err := pgSearch.RunGetManyQueryForSchema[storage.PolicyCategory](ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

// Count returns the number of objects in the store.
func (s *storeImpl) Count(ctx context.Context) (int, error) {

	var sacQueryFilter *v1.Query

	return pgSearch.RunCountRequestForSchema(ctx, schema, sacQueryFilter, s.db)
}

// Walk iterates through each policy category
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.PolicyCategory) error) error {
	var sacQueryFilter *v1.Query
	fetcher, closer, err := pgSearch.RunCursorQueryForSchema[storage.PolicyCategory](ctx, schema, sacQueryFilter, s.db)
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

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.PolicyCategory) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// UpsertMany saves the state of multiple objects in the storage.
func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.PolicyCategory) error {
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

//// Interface functions - END

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.PolicyCategory) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "PolicyCategory")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoPolicyCategories(ctx, batch, obj); err != nil {
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

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.PolicyCategory) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "PolicyCategory")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromPolicyCategories(ctx, tx, objs...); err != nil {
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

func (s *storeImpl) copyFromPolicyCategories(ctx context.Context, tx *postgres.Tx, objs ...*storage.PolicyCategory) error {

	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"name",

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

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"policy_categories"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func insertIntoPolicyCategories(_ context.Context, batch *pgx.Batch, obj *storage.PolicyCategory) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		serialized,
	}

	finalStr := "INSERT INTO policy_categories (Id, Name, serialized) VALUES($1, $2, $3) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}
