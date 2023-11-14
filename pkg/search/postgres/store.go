package postgres

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
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
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	batchAfter      = 100
	cursorBatchSize = 50
	deleteBatchSize = 5000

	// MaxBatchSize sets the maximum number of elements in a batch.
	// Using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	MaxBatchSize = 10000
)

var (
	errInvalidOperation = errors.New("invalid operation, function not set up")
)

// Deleter is an interface that allow deletions of multiple identifiers
type Deleter interface {
	DeleteMany(ctx context.Context, identifiers []string) error
}

type primaryKeyGetter[T any, PT unmarshaler[T]] func(obj PT) string
type durationTimeSetter func(start time.Time, op ops.Op)
type inserter[T any, PT unmarshaler[T]] func(batch *pgx.Batch, obj PT) error
type copier[T any, PT unmarshaler[T]] func(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...PT) error
type upsertChecker[T any, PT unmarshaler[T]] func(ctx context.Context, objs ...PT) error

func doNothingDurationTimeSetter(_ time.Time, _ ops.Op) {}

// GenericStore implements subset of Store interface for resources with single ID.
type GenericStore[T any, PT unmarshaler[T]] struct {
	mutex                            sync.RWMutex
	db                               postgres.DB
	schema                           *walker.Schema
	pkGetter                         primaryKeyGetter[T, PT]
	insertInto                       inserter[T, PT]
	copyFromObj                      copier[T, PT]
	setAcquireDBConnDuration         durationTimeSetter
	setPostgresOperationDurationTime durationTimeSetter
	permissionChecker                walker.PermissionChecker
	upsertAllowed                    upsertChecker[T, PT]
	targetResource                   permissions.ResourceMetadata
}

// NewGenericStore returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStore[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	upsertAllowed upsertChecker[T, PT],
	targetResource permissions.ResourceMetadata,
) *GenericStore[T, PT] {
	return &GenericStore[T, PT]{
		db:          db,
		schema:      schema,
		pkGetter:    pkGetter,
		insertInto:  insertInto,
		copyFromObj: copyFromObj,
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

// NewGenericStoreWithPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithPermissionChecker[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	checker walker.PermissionChecker,
) *GenericStore[T, PT] {
	return &GenericStore[T, PT]{
		db:          db,
		schema:      schema,
		pkGetter:    pkGetter,
		copyFromObj: copyFromObj,
		insertInto:  insertInto,
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
		permissionChecker: checker,
	}
}

// Exists tells whether the ID exists in the store.
func (s *GenericStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists)

	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()

	count, err := RunCountRequestForSchema(ctx, s.schema, q, s.db)
	// With joins and multiple paths to the scoping resources, it can happen that the Count query for an object identifier
	// returns more than 1, despite the fact that the identifier is unique in the table.
	return count > 0, err
}

// Count returns the number of objects in the store.
func (s *GenericStore[T, PT]) Count(ctx context.Context) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count)

	return RunCountRequestForSchema(ctx, s.schema, search.EmptyQuery(), s.db)
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *GenericStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Walk)

	fetcher, closer, err := RunCursorQueryForSchema[T, PT](ctx, s.schema, search.EmptyQuery(), s.db)
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

// GetAll retrieves all objects from the store.
//
// Deprecated: This can be dangerous on high cardinality stores consider Walk instead.
func (s *GenericStore[T, PT]) GetAll(ctx context.Context) ([]PT, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll)

	var objs []PT
	err := s.Walk(ctx, func(obj PT) error {
		objs = append(objs, obj)
		return nil
	})
	return objs, err
}

// Get returns the object, if it exists from the store.
func (s *GenericStore[T, PT]) Get(ctx context.Context, id string) (PT, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get)

	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()

	data, err := RunGetQueryForSchema[T, PT](ctx, s.schema, q, s.db)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	return data, true, nil
}

// GetByQuery returns the objects from the store matching the query.
func (s *GenericStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)

	rows, err := RunGetManyQueryForSchema[T, PT](ctx, s.schema, query, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rows, nil
}

// GetIDs returns all the IDs for the store.
func (s *GenericStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
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

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (s *GenericStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetMany)

	if len(identifiers) == 0 {
		return nil, nil, nil
	}

	q := search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery()

	rows, err := RunGetManyQueryForSchema[T, PT](ctx, s.schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			missingIndices := make([]int, 0, len(identifiers))
			for i := range identifiers {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	resultsByID := make(map[string]*T, len(rows))
	for _, msg := range rows {
		resultsByID[s.pkGetter(msg)] = msg
	}
	missingIndices := make([]int, 0, len(identifiers)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input identifiers
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*T, 0, len(resultsByID))
	for i, identifier := range identifiers {
		if result, ok := resultsByID[identifier]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (s *GenericStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)

	return RunDeleteRequestForSchema(ctx, s.schema, query, s.db)
}

// Delete removes the object associated to the specified ID from the store.
func (s *GenericStore[T, PT]) Delete(ctx context.Context, id string) error {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return s.DeleteByQuery(ctx, q)
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *GenericStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.RemoveMany)

	// Batch the deletes
	localBatchSize := deleteBatchSize
	numRecordsToDelete := len(identifiers)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		panic(err)
	}
	ctx = postgres.ContextWithTx(ctx, tx)

	for {
		if len(identifiers) == 0 {
			break
		}

		if len(identifiers) < localBatchSize {
			localBatchSize = len(identifiers)
		}

		identifierBatch := identifiers[:localBatchSize]

		q := search.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery()

		if err := RunDeleteRequestForSchema(ctx, s.schema, q, s.db); err != nil {
			return errors.Wrapf(err, "unable to delete the records.  Successfully deleted %d out of %d", numRecordsToDelete-len(identifiers), numRecordsToDelete)
		}

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	return tx.Commit(ctx)
}

// Upsert saves the current state of an object in storage.
func (s *GenericStore[T, PT]) Upsert(ctx context.Context, obj PT) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Upsert)

	if s.hasPermissionsChecker() {
		err := s.permissionCheckerAllowsUpsert(ctx)
		if err != nil {
			return err
		}
	} else if err := s.upsertAllowed(ctx, obj); err != nil {
		return err
	}

	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// UpsertMany saves the state of multiple objects in the storage.
func (s *GenericStore[T, PT]) UpsertMany(ctx context.Context, objs []PT) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.UpdateMany)

	if s.hasPermissionsChecker() {
		err := s.permissionCheckerAllowsUpsert(ctx)
		if err != nil {
			return err
		}
	} else if err := s.upsertAllowed(ctx, objs...); err != nil {
		return err
	}

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

		if s.copyFromObj == nil {
			return s.upsert(ctx, objs...)
		}

		return s.copyFrom(ctx, objs...)
	})
}

// region Helper functions

func (s *GenericStore[T, PT]) acquireConn(ctx context.Context, op ops.Op) (*postgres.Conn, error) {
	defer s.setAcquireDBConnDuration(time.Now(), op)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not acquire connection")
	}
	return conn, nil
}

func (s *GenericStore[T, PT]) hasPermissionsChecker() bool {
	return s.permissionChecker != nil
}

func (s *GenericStore[T, PT]) permissionCheckerAllowsUpsert(ctx context.Context) error {
	if !s.hasPermissionsChecker() {
		return utils.ShouldErr(errInvalidOperation)
	}
	allowed, err := s.permissionChecker.WriteAllowed(ctx)
	if err != nil {
		return err
	}
	if !allowed {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

func (s *GenericStore[T, PT]) upsert(ctx context.Context, objs ...PT) error {
	if s.insertInto == nil {
		return utils.ShouldErr(errInvalidOperation)
	}
	conn, err := s.acquireConn(ctx, ops.Upsert)
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, obj := range objs {
		batch := &pgx.Batch{}
		if err := s.insertInto(batch, obj); err != nil {
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

func (s *GenericStore[T, PT]) copyFrom(ctx context.Context, objs ...PT) error {
	if s.copyFromObj == nil {
		return utils.ShouldErr(errInvalidOperation)
	}

	conn, err := s.acquireConn(ctx, ops.UpsertAll)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "could not begin transaction")
	}

	if err := s.copyFromObj(ctx, s, tx, objs...); err != nil {
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

// GloballyScopedUpsertChecker returns upsertChecker for globally scoped objects
func GloballyScopedUpsertChecker[T any, PT unmarshaler[T]](targetResource permissions.ResourceMetadata) upsertChecker[T, PT] {
	return func(ctx context.Context, objs ...PT) error {
		scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
		if !scopeChecker.IsAllowed() {
			return sac.ErrResourceAccessDenied
		}
		return nil
	}
}

// endregion Helper functions
