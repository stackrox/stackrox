package postgres

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/contextutil"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// RowScanner scans a single pgx.Row into a proto message.
type RowScanner[T any] func(row pgx.Row) (*T, error)

// RowsScanner scans pgx.Rows into a slice of proto messages.
type RowsScanner[T any] func(rows pgx.Rows) ([]*T, error)

// FetchOption controls what data is fetched from the store.
type FetchOption func(*fetchConfig)

type fetchConfig struct {
	includeChildren bool
}

func defaultFetchConfig() fetchConfig {
	return fetchConfig{includeChildren: true}
}

func applyFetchOptions(opts []FetchOption) fetchConfig {
	cfg := defaultFetchConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithChildren explicitly requests child table data (the default).
func WithChildren() FetchOption {
	return func(c *fetchConfig) { c.includeChildren = true }
}

// WithoutChildren skips child table fetching for read performance.
// Repeated message fields will be returned as nil/empty slices.
func WithoutChildren() FetchOption {
	return func(c *fetchConfig) { c.includeChildren = false }
}

// NoSerializedStore is the store interface for types without a serialized column.
type NoSerializedStore[T any] interface {
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Walk(ctx context.Context, fn func(obj *T) error) error
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(obj *T) error) error
	Get(ctx context.Context, id string) (*T, bool, error)
	GetWithOptions(ctx context.Context, id string, opts ...FetchOption) (*T, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *T) error) error
	GetIDs(ctx context.Context) ([]string, error)
	GetIDsByQuery(ctx context.Context, query *v1.Query) ([]string, error)
	GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error)
	WalkByQueryWithOptions(ctx context.Context, q *v1.Query, fn func(obj *T) error, opts ...FetchOption) error
	DeleteByQuery(ctx context.Context, query *v1.Query) error
	DeleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error)
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, identifiers []string) error
	PruneMany(ctx context.Context, identifiers []string) error
	Upsert(ctx context.Context, obj *T) error
	UpsertMany(ctx context.Context, objs []*T) error
}

type noSerializedInserter[T any] func(batch *pgx.Batch, obj *T) error
type noSerializedCopier[T any] func(ctx context.Context, s Deleter, tx *postgres.Tx, objs ...*T) error
type noSerializedPKGetter[T any] func(obj *T) string
type noSerializedUpsertChecker[T any] func(ctx context.Context, objs ...*T) error

// ChildFetcher populates child table data on parent objects after the parent rows have been scanned.
type ChildFetcher[T any] func(ctx context.Context, q postgres.Queryable, objs []*T) error

type noSerializedGenericStore[T any] struct {
	mutex                            sync.RWMutex
	db                               postgres.DB
	schema                           *walker.Schema
	pkGetter                         noSerializedPKGetter[T]
	insertInto                       noSerializedInserter[T]
	copyFromObj                      noSerializedCopier[T]
	scanRow                          RowScanner[T]
	scanRows                         RowsScanner[T]
	childFetcher                     ChildFetcher[T]
	setAcquireDBConnDuration         durationTimeSetter
	setPostgresOperationDurationTime durationTimeSetter
	upsertAllowed                    noSerializedUpsertChecker[T]
	targetResource                   permissions.ResourceMetadata
	defaultSort                      *v1.QuerySortOption
	transformOptionsMap              search.OptionsMap
}

// NewNoSerializedGenericStore returns a new store for resources without a serialized column.
func NewNoSerializedGenericStore[T any](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter noSerializedPKGetter[T],
	insertInto noSerializedInserter[T],
	copyFromObj noSerializedCopier[T],
	scanRow RowScanner[T],
	scanRows RowsScanner[T],
	childFetcher ChildFetcher[T],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	upsertAllowed noSerializedUpsertChecker[T],
	targetResource permissions.ResourceMetadata,
	defaultSort *v1.QuerySortOption,
	transformOptionsMap search.OptionsMap,
) NoSerializedStore[T] {
	return &noSerializedGenericStore[T]{
		db:           db,
		schema:       schema,
		pkGetter:     pkGetter,
		insertInto:   insertInto,
		copyFromObj:  copyFromObj,
		scanRow:      scanRow,
		scanRows:     scanRows,
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
		upsertAllowed:       upsertAllowed,
		targetResource:      targetResource,
		defaultSort:         defaultSort,
		transformOptionsMap: transformOptionsMap,
	}
}

// NewNoSerializedGloballyScopedGenericStore returns a new store for globally-scoped resources without a serialized column.
func NewNoSerializedGloballyScopedGenericStore[T any](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter noSerializedPKGetter[T],
	insertInto noSerializedInserter[T],
	copyFromObj noSerializedCopier[T],
	scanRow RowScanner[T],
	scanRows RowsScanner[T],
	childFetcher ChildFetcher[T],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	targetResource permissions.ResourceMetadata,
	defaultSort *v1.QuerySortOption,
	transformOptionsMap search.OptionsMap,
) NoSerializedStore[T] {
	return &noSerializedGenericStore[T]{
		db:           db,
		schema:       schema,
		pkGetter:     pkGetter,
		insertInto:   insertInto,
		copyFromObj:  copyFromObj,
		scanRow:      scanRow,
		scanRows:     scanRows,
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
		upsertAllowed:       noSerializedGloballyScopedUpsertChecker[T](targetResource),
		targetResource:      targetResource,
		defaultSort:         defaultSort,
		transformOptionsMap: transformOptionsMap,
	}
}

func noSerializedGloballyScopedUpsertChecker[T any](targetResource permissions.ResourceMetadata) noSerializedUpsertChecker[T] {
	return func(ctx context.Context, objs ...*T) error {
		scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
		if !scopeChecker.IsAllowed() {
			return sac.ErrResourceAccessDenied
		}
		return nil
	}
}

func (s *noSerializedGenericStore[T]) applyQueryDefaults(q *v1.Query) *v1.Query {
	if s.transformOptionsMap != nil {
		q = sortfields.TransformSortOptions(q, s.transformOptionsMap)
	}
	if s.defaultSort == nil {
		return q
	}
	if q.GetPagination() == nil {
		q.Pagination = &v1.QueryPagination{}
	}
	if len(q.GetPagination().GetSortOptions()) == 0 {
		q.Pagination.SortOptions = []*v1.QuerySortOption{s.defaultSort}
	}
	return q
}

// Exists tells whether the ID exists in the store.
func (s *noSerializedGenericStore[T]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	count, err := RunCountRequestForSchema(ctx, s.schema, q, s.db)
	return count > 0, err
}

// Count returns the number of objects in the store.
func (s *noSerializedGenericStore[T]) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count)
	return RunCountRequestForSchema(ctx, s.schema, q, s.db)
}

// Search executes a search query against the store.
func (s *noSerializedGenericStore[T]) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Search)
	q = s.applyQueryDefaults(q)
	return RunSearchRequestForSchema(ctx, s.schema, q, s.db)
}

// Get returns the object, if it exists from the store.
func (s *noSerializedGenericStore[T]) Get(ctx context.Context, id string) (*T, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get)

	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	data, err := s.runGetQuery(ctx, s.schema, q)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	if err := s.fetchChildrenIfNeeded(ctx, s.db, []*T{data}); err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// GetWithOptions returns the object with configurable child table fetching.
func (s *noSerializedGenericStore[T]) GetWithOptions(ctx context.Context, id string, opts ...FetchOption) (*T, bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Get)

	cfg := applyFetchOptions(opts)
	schema := s.schemaForFetch(cfg)

	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	data, err := s.runGetQuery(ctx, schema, q)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	if cfg.includeChildren {
		if err := s.fetchChildrenIfNeeded(ctx, s.db, []*T{data}); err != nil {
			return nil, false, err
		}
	}
	return data, true, nil
}

func (s *noSerializedGenericStore[T]) schemaForFetch(cfg fetchConfig) *walker.Schema {
	if !cfg.includeChildren {
		return s.schema.ShallowCopyWithoutChildren()
	}
	return s.schema
}

func (s *noSerializedGenericStore[T]) fetchChildrenIfNeeded(ctx context.Context, q postgres.Queryable, objs []*T) error {
	if s.childFetcher == nil || len(objs) == 0 {
		return nil
	}
	return s.childFetcher(ctx, q, objs)
}

func (s *noSerializedGenericStore[T]) runGetQuery(ctx context.Context, schema *walker.Schema, q *v1.Query) (*T, error) {
	query, err := standardizeQueryAndPopulatePath(ctx, q, schema, GET)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, emptyQueryErr
	}

	var pool postgres.Queryable
	pool = s.db
	if tx, parentTxExists := postgres.TxFromContext(ctx); parentTxExists {
		pool = tx
	}

	return pgutils.Retry2(ctx, func() (*T, error) {
		row := tracedQueryRow(ctx, pool, query.AsSQL(), query.Data...)
		return s.scanRow(row)
	})
}

// GetByQueryFn iterates over all objects scoped by the query and applies the closure.
func (s *noSerializedGenericStore[T]) GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)
	query = s.applyQueryDefaults(query)
	return s.runQueryFn(ctx, query, fn)
}

func (s *noSerializedGenericStore[T]) runQueryFn(ctx context.Context, q *v1.Query, callback func(obj *T) error) error {
	var pool postgres.Queryable
	pool = s.db
	if tx, parentTxExists := postgres.TxFromContext(ctx); parentTxExists {
		pool = tx
	}

	rows, err := pgutils.Retry2(ctx, func() (*tracedRows, error) {
		return retryableGetRows(ctx, s.schema, q, pool)
	})
	if err != nil {
		return err
	}
	if rows == nil {
		return nil
	}

	results, err := s.scanRows(rows)
	if err != nil {
		return errors.Wrap(err, "scanning rows")
	}
	if err := s.fetchChildrenIfNeeded(ctx, pool, results); err != nil {
		return errors.Wrap(err, "fetching children")
	}
	for _, obj := range results {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "iterating over rows")
		}
		if err := callback(obj); err != nil {
			return err
		}
	}
	return nil
}

// GetByQuery returns the objects from the store matching the query.
func (s *noSerializedGenericStore[T]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)
	query = s.applyQueryDefaults(query)

	rows := make([]*T, 0, paginated.GetLimit(query.GetPagination().GetLimit(), batchAfter))
	err := s.runQueryFn(ctx, query, func(obj *T) error {
		rows = append(rows, obj)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows[0:len(rows):len(rows)], nil
}

func (s *noSerializedGenericStore[T]) walkByQuery(ctx context.Context, query *v1.Query, hint string, fn func(obj *T) error) error {
	query = s.applyQueryDefaults(query)
	return s.runCursorQueryFn(ctx, s.schema, query, hint, true, fn)
}

func (s *noSerializedGenericStore[T]) runCursorQueryFn(ctx context.Context, schema *walker.Schema, q *v1.Query, hint string, includeChildren bool, callback func(obj *T) error) error {
	ctx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, cursorDefaultTimeout)
	defer cancel()

	cursor, err := pgutils.Retry2(ctx, func() (*cursorSession, error) {
		return retryableGetCursorSession(ctx, schema, q, s.db, hint)
	})
	if err != nil {
		return errors.Wrap(err, "prepare cursor")
	}
	if cursor == nil {
		return nil
	}
	defer cursor.close()

	for {
		rows, err := cursor.tx.Query(ctx, fmt.Sprintf("FETCH %d FROM %s", cursorBatchSize, cursor.id))
		if err != nil {
			return errors.Wrap(err, "advancing in cursor")
		}

		results, scanErr := s.scanRows(rows)
		if scanErr != nil {
			return errors.Wrap(scanErr, "scanning cursor rows")
		}
		if includeChildren {
			if err := s.fetchChildrenIfNeeded(ctx, cursor.tx, results); err != nil {
				return errors.Wrap(err, "fetching children")
			}
		}

		for _, obj := range results {
			if ctx.Err() != nil {
				return errors.Wrap(ctx.Err(), "iterating over rows")
			}
			if err := callback(obj); err != nil {
				return errors.Wrap(err, "processing rows")
			}
		}

		if len(results) != cursorBatchSize {
			return ctx.Err()
		}
	}
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *noSerializedGenericStore[T]) Walk(ctx context.Context, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Walk)
	return s.walkByQuery(ctx, search.EmptyQuery(), "Walk", fn)
}

// WalkByQuery iterates over all the objects scoped by the query and applies the closure.
func (s *noSerializedGenericStore[T]) WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *T) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.WalkByQuery)
	return s.walkByQuery(ctx, query, "WalkByQuery", fn)
}

// WalkByQueryWithOptions iterates over objects with configurable child table fetching.
func (s *noSerializedGenericStore[T]) WalkByQueryWithOptions(ctx context.Context, query *v1.Query, fn func(obj *T) error, opts ...FetchOption) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.WalkByQuery)

	cfg := applyFetchOptions(opts)
	schema := s.schemaForFetch(cfg)
	query = s.applyQueryDefaults(query)
	return s.runCursorQueryFn(ctx, schema, query, "WalkByQueryWithOptions", cfg.includeChildren, fn)
}

func (s *noSerializedGenericStore[T]) fetchIDsByQuery(ctx context.Context, query *v1.Query) ([]string, error) {
	result, err := RunSearchRequestForSchema(ctx, s.schema, query, s.db)
	if err != nil {
		return nil, err
	}
	identifiers := make([]string, 0, len(result))
	for _, entry := range result {
		identifiers = append(identifiers, entry.ID)
	}
	return identifiers, nil
}

// GetIDs returns all the IDs for the store.
func (s *noSerializedGenericStore[T]) GetIDs(ctx context.Context) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetAll)
	return s.fetchIDsByQuery(ctx, search.EmptyQuery())
}

// GetIDsByQuery returns the IDs for the store matching the query.
func (s *noSerializedGenericStore[T]) GetIDsByQuery(ctx context.Context, query *v1.Query) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetByQuery)
	return s.fetchIDsByQuery(ctx, s.applyQueryDefaults(query))
}

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (s *noSerializedGenericStore[T]) GetMany(ctx context.Context, identifiers []string) ([]*T, []int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetMany)

	if len(identifiers) == 0 {
		return nil, nil, nil
	}

	q := search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery()

	resultsByID := make(map[string]*T, len(identifiers))
	err := s.runQueryFn(ctx, q, func(msg *T) error {
		resultsByID[s.pkGetter(msg)] = msg
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
	return elems, missingIndices, nil
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (s *noSerializedGenericStore[T]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	return RunDeleteRequestForSchema(ctx, s.schema, query, s.db)
}

// DeleteByQueryWithIDs removes the objects from the store based on the passed query returning deleted IDs.
func (s *noSerializedGenericStore[T]) DeleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	return RunDeleteRequestReturningIDsForSchema(ctx, s.schema, query, s.db)
}

// Delete removes the object associated to the specified ID from the store.
func (s *noSerializedGenericStore[T]) Delete(ctx context.Context, id string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	return RunDeleteRequestForSchema(ctx, s.schema, q, s.db)
}

// DeleteMany removes the objects associated to the specified IDs from the store within a transaction.
func (s *noSerializedGenericStore[T]) DeleteMany(ctx context.Context, identifiers []string) error {
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

	if err := s.deleteMany(ctx, identifiers, deleteBatchSize, false); err != nil {
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

// PruneMany removes the objects associated to the specified IDs from the store outside a transaction.
func (s *noSerializedGenericStore[T]) PruneMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Prune)
	return s.deleteMany(ctx, identifiers, pruneBatchSize, true)
}

func (s *noSerializedGenericStore[T]) deleteMany(ctx context.Context, identifiers []string, initialBatchSize int, continueOnError bool) error {
	deletedCount := 0
	numberToDelete := len(identifiers)

	if initialBatchSize <= 0 {
		return errors.New("batch size must be greater than 0")
	}

	for identifierBatch := range slices.Chunk(identifiers, initialBatchSize) {
		q := search.NewQueryBuilder().AddDocIDs(identifierBatch...).ProtoQuery()
		if err := RunDeleteRequestForSchema(ctx, s.schema, q, s.db); err != nil {
			if !continueOnError {
				return errors.Wrap(err, "unable to delete the records")
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				break
			}
			log.Errorf("unable to prune the records: %v", err)
		}
		deletedCount = deletedCount + len(identifierBatch)
		log.Debugf("deleted batch of %d records", len(identifierBatch))
	}
	log.Debugf("successfully deleted %d of %d records", deletedCount, numberToDelete)
	return nil
}

// Upsert saves the current state of an object in storage.
func (s *noSerializedGenericStore[T]) Upsert(ctx context.Context, obj *T) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Upsert)
	if err := s.upsertAllowed(ctx, obj); err != nil {
		return err
	}
	return pgutils.Retry(ctx, func() error {
		return s.upsert(ctx, obj)
	})
}

// UpsertMany saves the state of multiple objects in the storage.
func (s *noSerializedGenericStore[T]) UpsertMany(ctx context.Context, objs []*T) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.UpdateMany)
	if err := s.upsertAllowed(ctx, objs...); err != nil {
		return err
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

func (s *noSerializedGenericStore[T]) acquireConn(ctx context.Context, op ops.Op) (*postgres.Conn, error) {
	defer s.setAcquireDBConnDuration(time.Now(), op)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not acquire connection")
	}
	return conn, nil
}

func (s *noSerializedGenericStore[T]) upsert(ctx context.Context, objs ...*T) error {
	if s.insertInto == nil {
		return utils.ShouldErr(errInvalidOperation)
	}

	batch := &pgx.Batch{}
	for _, obj := range objs {
		if err := s.insertInto(batch, obj); err != nil {
			return errors.Wrap(err, "error on insertInto")
		}
	}

	if tx, parentTxExists := postgres.TxFromContext(ctx); parentTxExists {
		batchResults := postgres.BatchResultsFromPgx(tx.SendBatch(ctx, batch))
		if err := batchResults.Close(); err != nil {
			return errors.Wrap(err, "closing batch on transaction")
		}
		return nil
	}

	conn, err := s.acquireConn(ctx, ops.Upsert)
	if err != nil {
		return err
	}
	defer conn.Release()

	batchResults := conn.SendBatch(ctx, batch)
	if err := batchResults.Close(); err != nil {
		return errors.Wrap(err, "closing batch")
	}
	return nil
}

func (s *noSerializedGenericStore[T]) copyFrom(ctx context.Context, objs ...*T) error {
	if s.copyFromObj == nil {
		return utils.ShouldErr(errInvalidOperation)
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
