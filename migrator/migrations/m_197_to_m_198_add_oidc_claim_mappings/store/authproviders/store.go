package authproviders

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	schemaPkg "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/schema"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize       = 10000
	batchAfter      = 100
	deleteBatchSize = 5000
)

var (
	log    = logging.LoggerForModule()
	schema = schemaPkg.AuthProvidersSchema
)

type storeType = storage.AuthProvider

// Store is the interface to interact with the storage for storage.AuthProvider
type Store interface {
	UpsertMany(ctx context.Context, objs []*storeType) error
	Walk(ctx context.Context, fn func(obj *storeType) error) error
}

type storeImpl struct {
	mutex sync.RWMutex
	db    postgres.DB
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return &storeImpl{
		db: db,
	}
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storeType) error) error {
	fetcher, closer, err := pgSearch.RunCursorQueryForSchema[storage.AuthProvider, *storage.AuthProvider](ctx, schema, search.EmptyQuery(), s.db)
	if err != nil {
		return err
	}
	defer closer()
	for {
		rows, err := fetcher(batchSize)
		if err != nil {
			return pgutils.ErrNilIfNoRows(err)
		}
		for _, data := range rows {
			if err := fn(data); err != nil {
				return err
			}
		}
		if len(rows) != batchSize {
			break
		}
	}
	return nil
}

// UpsertMany saves the state of multiple objects in the storage.
func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storeType) error {
	return pgutils.Retry(func() error {
		// Lock since copyFrom requires a delete first before being executed.  If multiple processes are updating
		// same subset of rows, both deletes could occur before the copyFrom resulting in unique constraint
		// violations
		if len(objs) < batchAfter {
			s.mutex.RLock()
			defer s.mutex.RUnlock()

			return s.upsert(ctx, objs...)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()

		return s.copyFrom(ctx, objs...)
	})
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.AuthProvider) error {
	conn, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "could not begin transaction")
	}

	if err := copyFromAuthProviders(ctx, s, tx, objs...); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Wrap(rollbackErr, "could not rollback transaction")
		}
		return errors.Wrap(err, "copy from objects failed")
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "could not commit transaction")
	}
	return nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *storeImpl) DeleteMany(ctx context.Context, identifiers []string) error {
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
		q := search.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery()

		if err := pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db); err != nil {
			return errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(identifiers), numRecordsToDelete)
		}

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	return nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.AuthProvider) error {
	conn, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := insertIntoAuthProviders(batch, obj); err != nil {
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

func (s *storeImpl) acquireConn(ctx context.Context) (*postgres.Conn, error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not acquire connection")
	}
	return conn, nil
}

func insertIntoAuthProviders(batch *pgx.Batch, obj *storage.AuthProvider) error {
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

	finalStr := "INSERT INTO auth_providers (Id, Name, serialized) VALUES($1, $2, $3) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromAuthProviders(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.AuthProvider) error {
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

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
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"auth_providers"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}
