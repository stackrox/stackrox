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
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	batchAfter      = 100
	cursorBatchSize = 50
	deleteBatchSize = 65000

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
	cache                            map[string]PT
	useCache                         bool
	cacheLock                        sync.RWMutex
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
		cache:          nil,
		useCache:       false,
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
		cache:             nil,
		useCache:          false,
	}
}

// NewGenericStoreWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithCache[T any, PT unmarshaler[T]](
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
	store := &GenericStore[T, PT]{
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
		cache:          make(map[string]PT),
		useCache:       true,
	}
	store.resetCache(sac.WithAllAccess(context.Background()))
	return store
}

// NewGenericStoreWithPermissionCheckerWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithPermissionCheckerWithCache[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	checker walker.PermissionChecker,
) *GenericStore[T, PT] {
	store := &GenericStore[T, PT]{
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
		cache:             make(map[string]PT),
		useCache:          true,
	}
	store.resetCache(sac.WithAllAccess(context.Background()))
	return store
}

// Exists tells whether the ID exists in the store.
func (s *GenericStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Exists)

	if s.useCache {
		return s.existsInCache(ctx, id)
	}

	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()

	count, err := RunCountRequestForSchema(ctx, s.schema, q, s.db)
	// With joins and multiple paths to the scoping resources, it can happen that the Count query for an object identifier
	// returns more than 1, despite the fact that the identifier is unique in the table.
	return count > 0, err
}

// Count returns the number of objects in the store.
func (s *GenericStore[T, PT]) Count(ctx context.Context) (int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Count)

	if s.useCache {
		return s.countFromCache(ctx)
	}

	return RunCountRequestForSchema(ctx, s.schema, search.EmptyQuery(), s.db)
}

// Walk iterates over all the objects in the store and applies the closure.
func (s *GenericStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Walk)

	if s.useCache {
		return s.walkCache(ctx, fn)
	}
	return s.doDBWalk(ctx, fn)
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

	if s.useCache {
		return s.getFromCache(ctx, id)
	}

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

	if s.useCache {
		return s.getIDsFromCache(ctx)
	}

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
func (s *GenericStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]PT, []int, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.GetMany)

	if len(identifiers) == 0 {
		return nil, nil, nil
	}

	if s.useCache {
		return s.getManyFromCache(ctx, identifiers)
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
	resultsByID := make(map[string]PT, len(rows))
	for _, msg := range rows {
		resultsByID[s.pkGetter(msg)] = msg
	}
	missingIndices := make([]int, 0, len(identifiers)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input identifiers
	// slice, since some calling code relies on that to maintain order.
	elems := make([]PT, 0, len(resultsByID))
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
func (s *GenericStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) ([]string, error) {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)

	return RunDeleteRequestReturningIDsForSchema(ctx, s.schema, query, s.db)
}

// Delete removes the object associated to the specified ID from the store.
func (s *GenericStore[T, PT]) Delete(ctx context.Context, id string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.Remove)
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	dbErr := RunDeleteRequestForSchema(ctx, s.schema, q, s.db)
	if dbErr != nil {
		s.resetCache(ctx)
		return dbErr
	}
	s.removeFromCache(id)
	return nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (s *GenericStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	defer s.setPostgresOperationDurationTime(time.Now(), ops.RemoveMany)

	// Batch the deletes
	localBatchSize := deleteBatchSize
	var err error
	var tx *postgres.Tx
	if !postgres.HasTxInContext(ctx) {
		tx, err = s.db.Begin(ctx)
		if err != nil {
			return errors.Wrap(err, "could not create transaction for deletes")
		}

		ctx = postgres.ContextWithTx(ctx, tx)
	}

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
			if tx != nil {
				if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
					return errors.Wrapf(err, "unable to delete records and rollback failed: %v", rollbackErr)
				}
			}
			s.resetCache(ctx)
			return errors.Wrap(err, "unable to delete the records")
		}
		s.removeManyFromCache(identifierBatch)

		// Move the slice forward to start the next batch
		identifiers = identifiers[localBatchSize:]
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
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

	dbErr := pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
	if dbErr != nil {
		s.resetCache(ctx)
		return dbErr
	}
	s.pushToCache(obj)
	return nil
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

	dbErr := pgutils.Retry(func() error {
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
	if dbErr != nil {
		s.resetCache(ctx)
		return dbErr
	}
	s.pushManyToCache(objs)
	return nil
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

	batch := &pgx.Batch{}
	for _, obj := range objs {
		if err := s.insertInto(batch, obj); err != nil {
			return errors.Wrap(err, "error on insertInto")
		}
	}
	batchResults := conn.SendBatch(ctx, batch)
	if err := batchResults.Close(); err != nil {
		return errors.Wrap(err, "closing batch")
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

func (s *GenericStore[T, PT]) doDBWalk(ctx context.Context, walkFn func(PT) error) error {
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
			if err := walkFn(data); err != nil {
				return err
			}
		}
		if len(rows) != cursorBatchSize {
			break
		}
	}
	return nil
}

func (s *GenericStore[T, PT]) resetCache(ctx context.Context) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	unrestrictedCtx := sac.WithAllAccess(ctx)
	s.cache = make(map[string]PT)
	_ = s.doDBWalk(unrestrictedCtx, func(obj PT) error {
		s.pushToCacheNoLock(obj)
		return nil
	})
}

func (s *GenericStore[T, PT]) pushManyToCache(objs []PT) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	for _, obj := range objs {
		s.pushToCacheNoLock(obj)
	}
}

func (s *GenericStore[T, PT]) pushToCache(obj PT) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.cache[s.pkGetter(obj)] = obj
}

func (s *GenericStore[T, PT]) pushToCacheNoLock(obj PT) {
	s.cache[s.pkGetter(obj)] = obj
}

func (s *GenericStore[T, PT]) removeManyFromCache(ids []string) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	for _, id := range ids {
		s.removeFromCacheNoLock(id)
	}
}

func (s *GenericStore[T, PT]) removeFromCache(id string) {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	delete(s.cache, id)
}

func (s *GenericStore[T, PT]) removeFromCacheNoLock(id string) {
	delete(s.cache, id)
}

func (s *GenericStore[T, PT]) getFromCache(ctx context.Context, id string) (PT, bool, error) {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	val, found := s.cache[id]
	found = found && s.isReadAllowed(ctx, val)
	return val, found, nil
}

func (s *GenericStore[T, PT]) getManyFromCache(ctx context.Context, identifiers []string) ([]PT, []int, error) {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	results := make([]PT, 0, len(identifiers))
	misses := make([]int, 0, len(identifiers))
	for ix, id := range identifiers {
		val, found := s.cache[id]
		found = found && s.isReadAllowed(ctx, val)
		if found {
			results = append(results, val)
		} else {
			misses = append(misses, ix)
		}
	}
	return results, misses, nil
}

func (s *GenericStore[T, PT]) getIDsFromCache(ctx context.Context) ([]string, error) {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	results := make([]string, 0, len(s.cache))
	for k, v := range s.cache {
		if !s.isReadAllowed(ctx, v) {
			continue
		}
		results = append(results, k)
	}
	return results, nil
}

func (s *GenericStore[T, PT]) getAllFromCache(ctx context.Context) ([]PT, error) {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	results := make([]PT, 0, len(s.cache))
	for _, v := range s.cache {
		if !s.isReadAllowed(ctx, v) {
			continue
		}
		results = append(results, v)
	}
	return results, nil
}

func (s *GenericStore[T, PT]) existsInCache(ctx context.Context, id string) (bool, error) {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	val, found := s.cache[id]
	found = found && s.isReadAllowed(ctx, val)
	return found, nil
}

func (s *GenericStore[T, PT]) walkCache(ctx context.Context, walkFn func(obj PT) error) error {
	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()
	for _, v := range s.cache {
		if !s.isReadAllowed(ctx, v) {
			continue
		}
		err := walkFn(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *GenericStore[T, PT]) countFromCache(ctx context.Context) (int, error) {
	count := 0
	err := s.walkCache(ctx, func(_ PT) error {
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *GenericStore[T, PT]) isReadAllowed(ctx context.Context, obj PT) bool {
	if s.hasPermissionsChecker() {
		allowed, err := s.permissionChecker.ReadAllowed(ctx)
		if err != nil {
			return false
		}
		return allowed
	}
	scopeChecker := sac.GlobalAccessScopeChecker(ctx)
	scopeChecker = scopeChecker.AccessMode(storage.Access_READ_ACCESS)
	scopeChecker = scopeChecker.Resource(s.targetResource)
	switch s.targetResource.GetScope() {
	case permissions.NamespaceScope:
		var interfaceObj interface{}
		interfaceObj = obj
		namespaceScopedObj := interfaceObj.(sac.NamespaceScopedObject)
		scopeChecker = scopeChecker.ForNamespaceScopedObject(namespaceScopedObj)
	case permissions.ClusterScope:
		var interfaceObj interface{}
		interfaceObj = obj
		clusterScopedObj := interfaceObj.(sac.ClusterScopedObject)
		scopeChecker = scopeChecker.ForClusterScopedObject(clusterScopedObj)
	}
	return scopeChecker.IsAllowed()
}

// endregion Helper functions
