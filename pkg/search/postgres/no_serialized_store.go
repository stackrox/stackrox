package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sync"
)

// NoSerializedStore is a store interface for types that don't use serialized blobs.
// It is identical to Store but with less restrictive type constraints.
type NoSerializedStore[T any] interface {
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Walk(ctx context.Context, fn func(obj *T) error) error
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(obj *T) error) error
	Get(ctx context.Context, id string) (*T, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *T) error) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error)
	DeleteByQuery(ctx context.Context, query *v1.Query) error
	DeleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error)
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, identifiers []string) error
	PruneMany(ctx context.Context, identifiers []string) error
	Upsert(ctx context.Context, obj *T) error
	UpsertMany(ctx context.Context, objs []*T) error
}

// ChildFetcher fetches child table data for a set of parent objects, keyed by parent ID.
type ChildFetcher[T any] func(ctx context.Context, db postgres.DB, objs []*T) error

// BulkInserter queues unnest-based bulk INSERT statements into a pgx.Batch for all objects at once.
// This is much more efficient than per-row inserts for large batches because Postgres
// parses and plans fewer statements.
type BulkInserter[T any] func(batch *pgx.Batch, objs []*T) error

type noSerializedStore[T any] struct {
	db                               postgres.DB
	schema                           *walker.Schema
	pkGetter                         func(obj *T) string
	insertInto                       func(batch *pgx.Batch, obj *T) error
	bulkInsert                       BulkInserter[T]
	copyFromObj                      func(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*T) error
	rowScanner                       RowScanner[T]
	rowsScanner                      RowsScanner[T]
	childFetcher                     ChildFetcher[T]
	setAcquireDBConnDuration         durationTimeSetter
	setPostgresOperationDurationTime durationTimeSetter
	upsertAllowed                    func(ctx context.Context, objs ...*T) error
	targetResource                   permissions.ResourceMetadata
	mutex                            sync.RWMutex
}

// NoSerializedStoreOpts holds optional configuration for NewNoSerializedStore.
type NoSerializedStoreOpts[T any] struct {
	ChildFetcher ChildFetcher[T]
	BulkInsert   BulkInserter[T]
}

// NewNoSerializedStore creates a store that uses column-based scanning instead of serialized blobs.
func NewNoSerializedStore[T any](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter func(obj *T) string,
	insertInto func(batch *pgx.Batch, obj *T) error,
	copyFromObj func(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*T) error,
	rowScanner RowScanner[T],
	rowsScanner RowsScanner[T],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	upsertAllowed func(ctx context.Context, objs ...*T) error,
	targetResource permissions.ResourceMetadata,
	opts ...NoSerializedStoreOpts[T],
) NoSerializedStore[T] {
	var childFetcher ChildFetcher[T]
	var bulkInsert BulkInserter[T]
	if len(opts) > 0 {
		childFetcher = opts[0].ChildFetcher
		bulkInsert = opts[0].BulkInsert
	}
	if upsertAllowed == nil {
		upsertAllowed = func(ctx context.Context, objs ...*T) error {
			scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
			if !scopeChecker.IsAllowed() {
				return sac.ErrResourceAccessDenied
			}
			return nil
		}
	}
	return &noSerializedStore[T]{
		db:           db,
		schema:       schema,
		pkGetter:     pkGetter,
		insertInto:   insertInto,
		bulkInsert:   bulkInsert,
		copyFromObj:  copyFromObj,
		rowScanner:   rowScanner,
		rowsScanner:  rowsScanner,
		childFetcher: childFetcher,
		setAcquireDBConnDuration: func() durationTimeSetter {
			if setAcquireDBConnDuration == nil {
				return doNothingDurationTimeSetter
			}
			return setAcquireDBConnDuration
		}(),
		setPostgresOperationDurationTime: func() durationTimeSetter {
			if setPostgresOperationDurationTime == nil {
				return doNothingDurationTimeSetter
			}
			return setPostgresOperationDurationTime
		}(),
		upsertAllowed:  upsertAllowed,
		targetResource: targetResource,
	}
}

func (s *noSerializedStore[T]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	count, err := RunCountRequestForSchema(ctx, s.schema, q, s.db)
	return count > 0, err
}

func (s *noSerializedStore[T]) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count)
	return RunCountRequestForSchema(ctx, s.schema, q, s.db)
}

func (s *noSerializedStore[T]) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Search)
	return RunSearchRequestForSchema(ctx, s.schema, q, s.db)
}

func (s *noSerializedStore[T]) Walk(ctx context.Context, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Walk)
	return RunCursorQueryForSchemaFnWithScanner(ctx, s.schema, search.EmptyQuery(), s.db, s.rowsScanner, fn)
}

func (s *noSerializedStore[T]) WalkByQuery(ctx context.Context, q *v1.Query, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.WalkByQuery)
	return RunCursorQueryForSchemaFnWithScanner(ctx, s.schema, q, s.db, s.rowsScanner, fn)
}

func (s *noSerializedStore[T]) fetchChildren(ctx context.Context, objs ...*T) error {
	if s.childFetcher == nil || len(objs) == 0 {
		return nil
	}
	return s.childFetcher(ctx, s.db, objs)
}

func (s *noSerializedStore[T]) Get(ctx context.Context, id string) (*T, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	data, err := RunGetQueryForSchemaWithScanner(ctx, s.schema, q, s.db, s.rowScanner)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	if err := s.fetchChildren(ctx, data); err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s *noSerializedStore[T]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)
	rows := make([]*T, 0, paginated.GetLimit(query.GetPagination().GetLimit(), batchAfter))
	err := RunQueryForSchemaFnWithScanner(ctx, s.schema, query, s.db, s.rowsScanner, func(obj *T) error {
		rows = append(rows, obj)
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := rows[0:len(rows):len(rows)]
	if err := s.fetchChildren(ctx, result...); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *noSerializedStore[T]) GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)
	return RunQueryForSchemaFnWithScanner(ctx, s.schema, query, s.db, s.rowsScanner, fn)
}

func (s *noSerializedStore[T]) GetIDs(ctx context.Context) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll)
	result, err := RunSearchRequestForSchema(ctx, s.schema, search.EmptyQuery(), s.db)
	if err != nil {
		return nil, err
	}
	identifiers := make([]string, 0, len(result))
	for _, entry := range result {
		identifiers = append(identifiers, entry.ID)
	}
	return identifiers, nil
}

func (s *noSerializedStore[T]) GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetMany)
	if len(identifiers) == 0 {
		return nil, nil, nil
	}
	q := search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery()
	resultsByID := make(map[string]*T, len(identifiers))
	err := RunQueryForSchemaFnWithScanner(ctx, s.schema, q, s.db, s.rowsScanner, func(obj *T) error {
		resultsByID[s.pkGetter(obj)] = obj
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	missingIndices := make([]int, 0, len(identifiers)-len(resultsByID))
	elems := make([]*T, 0, len(resultsByID))
	for i, identifier := range identifiers {
		if result, ok := resultsByID[identifier]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	if err := s.fetchChildren(ctx, elems...); err != nil {
		return nil, nil, err
	}
	return elems, missingIndices, nil
}

func (s *noSerializedStore[T]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	return RunDeleteRequestForSchema(ctx, s.schema, query, s.db)
}

func (s *noSerializedStore[T]) DeleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	return RunDeleteRequestReturningIDsForSchema(ctx, s.schema, query, s.db)
}

func (s *noSerializedStore[T]) Delete(ctx context.Context, id string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return RunDeleteRequestForSchema(ctx, s.schema, q, s.db)
}

func (s *noSerializedStore[T]) DeleteMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.RemoveMany)

	var err error
	var tx *postgres.Tx
	if !postgres.HasTxInContext(ctx) {
		tx, err = s.db.Begin(ctx)
		if err != nil {
			return errors.Wrap(err, "could not create transaction for deletes")
		}
		ctx = postgres.ContextWithTx(ctx, tx)
	}

	if err := s.deleteMany(ctx, identifiers, deleteBatchSize); err != nil {
		if tx != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return errors.Wrapf(err, "unable to delete records and rollback failed: %v", rollbackErr)
			}
		}
		return err
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
}

func (s *noSerializedStore[T]) PruneMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Prune)
	return s.deleteMany(ctx, identifiers, pruneBatchSize)
}

func (s *noSerializedStore[T]) deleteMany(ctx context.Context, identifiers []string, batchSize int) error {
	for i := 0; i < len(identifiers); i += batchSize {
		end := i + batchSize
		if end > len(identifiers) {
			end = len(identifiers)
		}
		q := search.NewQueryBuilder().AddDocIDs(identifiers[i:end]...).ProtoQuery()
		if err := RunDeleteRequestForSchema(ctx, s.schema, q, s.db); err != nil {
			return errors.Wrap(err, "unable to delete the records")
		}
	}
	return nil
}

func (s *noSerializedStore[T]) Upsert(ctx context.Context, obj *T) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Upsert)
	if s.upsertAllowed != nil {
		if err := s.upsertAllowed(ctx, obj); err != nil {
			return err
		}
	}
	return pgutils.Retry(ctx, func() error {
		return s.upsert(ctx, obj)
	})
}

func (s *noSerializedStore[T]) UpsertMany(ctx context.Context, objs []*T) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.UpdateMany)
	if s.upsertAllowed != nil {
		if err := s.upsertAllowed(ctx, objs...); err != nil {
			return err
		}
	}
	return pgutils.Retry(ctx, func() error {
		if len(objs) < batchAfter || s.copyFromObj == nil {
			s.mutex.RLock()
			defer s.mutex.RUnlock()
			return s.upsert(ctx, objs...)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()
		return s.copyFrom(ctx, objs...)
	})
}

func (s *noSerializedStore[T]) acquireConn(ctx context.Context, op ops.Op) (*postgres.Conn, error) {
	defer s.setAcquireDBConnDuration(time.Now(), op)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not acquire connection")
	}
	return conn, nil
}

func (s *noSerializedStore[T]) upsert(ctx context.Context, objs ...*T) error {
	if s.insertInto == nil {
		return errors.New("insert function not set")
	}
	batch := &pgx.Batch{}
	if s.bulkInsert != nil && len(objs) > 1 {
		if err := s.bulkInsert(batch, objs); err != nil {
			return errors.Wrap(err, "error on bulkInsert")
		}
	} else {
		for _, obj := range objs {
			if err := s.insertInto(batch, obj); err != nil {
				return errors.Wrap(err, "error on insertInto")
			}
		}
	}
	if tx, parentTxExists := postgres.TxFromContext(ctx); parentTxExists {
		batchResults := postgres.BatchResultsFromPgx(tx.SendBatch(ctx, batch))
		return batchResults.Close()
	}
	conn, err := s.acquireConn(ctx, ops.Upsert)
	if err != nil {
		return err
	}
	defer conn.Release()
	batchResults := conn.SendBatch(ctx, batch)
	return batchResults.Close()
}

func (s *noSerializedStore[T]) copyFrom(ctx context.Context, objs ...*T) error {
	if s.copyFromObj == nil {
		return errors.New("copyFrom function not set")
	}
	tx, parentTxExists := postgres.TxFromContext(ctx)
	if !parentTxExists {
		conn, err := s.acquireConn(ctx, ops.UpsertAll)
		if err != nil {
			return err
		}
		defer conn.Release()
		var txErr error
		tx, ctx, txErr = conn.Begin(ctx)
		if txErr != nil {
			return errors.Wrap(txErr, "could not begin transaction")
		}
	}
	if err := s.copyFromObj(ctx, s, tx, objs...); err != nil {
		if !parentTxExists {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return errors.Wrap(rollbackErr, "could not rollback transaction")
			}
		}
		return errors.Wrap(err, "copy from objects failed")
	}
	if !parentTxExists {
		if err := tx.Commit(ctx); err != nil {
			return errors.Wrap(err, "could not commit transaction")
		}
	}
	return nil
}
